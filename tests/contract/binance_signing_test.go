// Package contract_test verifies the wire-level contracts this codebase
// depends on but does not control: how the Binance broker adapter signs
// requests, and the shape the Quant Engine fake promises to
// application/ports/output.QuantEngine callers.
package contract_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"trading-core/internal/infrastructure/external/broker/binance"
)

const testAPISecret = "test-secret-value"

func TestBinanceClientSignsRequestsWithHMACSHA256(t *testing.T) {
	var capturedQuery url.Values
	var capturedAPIKeyHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		capturedAPIKeyHeader = r.Header.Get("X-MBX-APIKEY")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalWalletBalance":"1000","totalMarginBalance":"1000","availableBalance":"1000"}`))
	}))
	defer server.Close()

	client := binance.NewClient(server.URL, "test-api-key", testAPISecret, 5000, 5*time.Second)
	broker := binance.NewBroker(client)

	if _, err := broker.GetAccount(context.Background()); err != nil {
		t.Fatalf("GetAccount: %v", err)
	}

	if capturedAPIKeyHeader != "test-api-key" {
		t.Fatalf("got X-MBX-APIKEY header %q, want %q", capturedAPIKeyHeader, "test-api-key")
	}

	gotSignature := capturedQuery.Get("signature")
	if gotSignature == "" {
		t.Fatal("request was not signed: missing signature query parameter")
	}
	if capturedQuery.Get("timestamp") == "" {
		t.Fatal("request is missing the required timestamp parameter")
	}
	if capturedQuery.Get("recvWindow") != "5000" {
		t.Fatalf("got recvWindow %q, want \"5000\"", capturedQuery.Get("recvWindow"))
	}

	// Recompute the expected signature the same way the client does: HMAC-
	// SHA256 over every OTHER parameter, in the exact query string the
	// server received (order matters for the signature but not for this
	// re-derivation, since we sign the same encoded string back).
	unsigned := url.Values{}
	for k, v := range capturedQuery {
		if k == "signature" {
			continue
		}
		unsigned[k] = v
	}
	mac := hmac.New(sha256.New, []byte(testAPISecret))
	mac.Write([]byte(unsigned.Encode()))
	wantSignature := hex.EncodeToString(mac.Sum(nil))

	if gotSignature != wantSignature {
		t.Fatalf("got signature %s, want %s (request forgeable or secret not applied correctly)", gotSignature, wantSignature)
	}
}

// TestBinanceAPIErrorClassifiesOrderNotFoundCodes locks in the contract
// broker.go's GetOrder/FindOrderByClientOrderID rely on: Binance's
// "order does not exist" family of error codes must map to IsNotFound so
// they can be translated into output.ErrNotFound.
func TestBinanceAPIErrorClassifiesOrderNotFoundCodes(t *testing.T) {
	cases := []struct {
		code       int
		wantResult bool
	}{
		{-2013, true},  // "Order does not exist"
		{-2011, true},  // "Unknown order sent"
		{-1021, false}, // "Timestamp outside of recvWindow" — not a not-found case
		{0, false},
	}
	for _, tc := range cases {
		apiErr := &binance.APIError{StatusCode: http.StatusBadRequest, Code: tc.code, Msg: "test"}
		if got := apiErr.IsNotFound(); got != tc.wantResult {
			t.Errorf("code %d: IsNotFound() = %v, want %v", tc.code, got, tc.wantResult)
		}
	}
}
