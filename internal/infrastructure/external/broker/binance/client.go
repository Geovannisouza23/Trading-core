// Package binance implements a REST client for the Binance Futures
// (USDⓈ-M) API, shared by the testnet and real broker adapters — they only
// differ in base URL and the extra safety gates the real adapter applies.
package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker"
)

// APIError represents a structured error response from Binance
// ({"code": -2013, "msg": "..."}).
type APIError struct {
	StatusCode int
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("binance: http %d code %d: %s", e.StatusCode, e.Code, e.Msg)
}

// IsNotFound reports whether the error corresponds to Binance's "order does
// not exist" family of error codes.
func (e *APIError) IsNotFound() bool {
	return e.Code == -2013 || e.Code == -2011
}

// Client is a minimal, signed REST client covering exactly the endpoints
// the Broker port needs — nothing more.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	apiSecret  string
	recvWindow int
	breaker    *gobreaker.CircuitBreaker
}

// NewClient builds a Client. baseURL differs between testnet and real; the
// signing scheme and endpoint set are identical.
func NewClient(baseURL, apiKey, apiSecret string, recvWindowMs int, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		recvWindow: recvWindowMs,
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "binance-broker",
			Timeout: 30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > 5
			},
		}),
	}
}

func (c *Client) sign(params url.Values) {
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", strconv.Itoa(c.recvWindow))
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(params.Encode()))
	params.Set("signature", hex.EncodeToString(mac.Sum(nil)))
}

// do executes a signed request through the circuit breaker and decodes the
// JSON response into out (if non-nil).
func (c *Client) do(ctx context.Context, method, path string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}
	c.sign(params)

	_, err := c.breaker.Execute(func() (any, error) {
		var req *http.Request
		var err error
		switch method {
		case http.MethodGet, http.MethodDelete:
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path+"?"+params.Encode(), nil)
		default:
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, strings.NewReader(params.Encode()))
			if req != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		}
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-MBX-APIKEY", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			_ = json.Unmarshal(body, apiErr)
			return nil, apiErr
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return nil, fmt.Errorf("decoding binance response: %w", err)
			}
		}
		return nil, nil
	})
	return err
}
