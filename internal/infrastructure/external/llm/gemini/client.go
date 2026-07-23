// Package gemini implements output.EventIntelligence against the Gemini
// API using structured output. It is inert without an API key (LLMConfig
// defaults Provider to "noop"); when enabled, it enforces a timeout,
// payload size limit, and strict schema validation on the response, and it
// never has access to any tool, broker, or order-related capability.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/shared"
)

type Client struct {
	httpClient      *http.Client
	baseURL         string
	apiKey          string
	model           string
	maxPayloadBytes int
}

func NewClient(baseURL, apiKey, model string, timeout time.Duration, maxPayloadBytes int) *Client {
	return &Client{
		httpClient:      &http.Client{Timeout: timeout},
		baseURL:         baseURL,
		apiKey:          apiKey,
		model:           model,
		maxPayloadBytes: maxPayloadBytes,
	}
}

var _ output.EventIntelligence = (*Client)(nil)

func (c *Client) Analyze(ctx context.Context, input output.NewsAnalysisInput) (output.EventAssessment, error) {
	newsContent := input.Content
	if len(newsContent) > c.maxPayloadBytes {
		newsContent = newsContent[:c.maxPayloadBytes]
	}

	prompt := fmt.Sprintf(
		"Source: %s\nURL: %s\nTitle: %s\nPublished: %s\n\nContent:\n%s",
		input.Source, input.URL, input.Title, input.PublishedAt.Format(time.RFC3339), newsContent,
	)

	reqBody := generateContentRequest{
		Contents: []content{{
			Role:  "user",
			Parts: []part{{Text: prompt}},
		}},
		SystemInstruction: &content{Parts: []part{{Text: systemInstruction}}},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   responseSchema,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return output.EventAssessment{}, fmt.Errorf("encoding gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return output.EventAssessment{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return output.EventAssessment{}, fmt.Errorf("calling gemini: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return output.EventAssessment{}, err
	}
	if resp.StatusCode >= 400 {
		return output.EventAssessment{}, fmt.Errorf("gemini returned http %d: %s", resp.StatusCode, string(body))
	}

	var apiResp generateContentResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return output.EventAssessment{}, fmt.Errorf("decoding gemini response envelope: %w", err)
	}
	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return output.EventAssessment{}, fmt.Errorf("gemini returned no candidates")
	}

	var parsed assessmentPayload
	if err := json.Unmarshal([]byte(apiResp.Candidates[0].Content.Parts[0].Text), &parsed); err != nil {
		return output.EventAssessment{}, fmt.Errorf("gemini structured output is not valid JSON: %w", err)
	}

	return toEventAssessment(parsed)
}

func toEventAssessment(p assessmentPayload) (output.EventAssessment, error) {
	direction := event.Direction(p.Direction)
	severity := event.Severity(p.Severity)
	action := event.Action(p.Action)
	if !action.Valid() {
		return output.EventAssessment{}, fmt.Errorf("gemini returned an unknown action %q", p.Action)
	}
	switch direction {
	case event.DirectionBullish, event.DirectionBearish, event.DirectionNeutral:
	default:
		return output.EventAssessment{}, fmt.Errorf("gemini returned an unknown direction %q", p.Direction)
	}
	switch severity {
	case event.SeverityLow, event.SeverityMedium, event.SeverityHigh, event.SeverityCritical:
	default:
		return output.EventAssessment{}, fmt.Errorf("gemini returned an unknown severity %q", p.Severity)
	}
	if len(p.AffectedAssets) == 0 {
		return output.EventAssessment{}, fmt.Errorf("gemini returned no affected assets")
	}

	confidence, err := shared.NewConfidence(decimal.NewFromFloat(p.Confidence))
	if err != nil {
		return output.EventAssessment{}, fmt.Errorf("gemini returned an invalid confidence: %w", err)
	}

	assets := make([]shared.Symbol, 0, len(p.AffectedAssets))
	for _, raw := range p.AffectedAssets {
		symbol, err := shared.NewSymbol(raw)
		if err != nil {
			return output.EventAssessment{}, fmt.Errorf("gemini returned an invalid symbol %q: %w", raw, err)
		}
		assets = append(assets, symbol)
	}

	return output.EventAssessment{
		EventType:         p.EventType,
		Direction:         direction,
		Severity:          severity,
		Confidence:        confidence,
		AffectedAssets:    assets,
		Action:            action,
		SourceCount:       p.SourceCount,
		HasOfficialSource: p.HasOfficialSource,
	}, nil
}
