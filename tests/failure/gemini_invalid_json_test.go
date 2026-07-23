package failure_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/infrastructure/external/llm/gemini"
)

func TestGeminiClientRejectsInvalidStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The outer envelope is valid, but the inner structured-output text
		// is not valid JSON — this must be rejected, not silently accepted.
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"not json at all"}]}}]}`))
	}))
	defer server.Close()

	client := gemini.NewClient(server.URL, "test-key", "gemini-2.5-flash", 5*time.Second, 16*1024)

	_, err := client.Analyze(context.Background(), output.NewsAnalysisInput{
		Source: "reuters.com", URL: "https://reuters.com/a", Title: "t", Content: "c", PublishedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for non-JSON structured output, got nil")
	}
}

func TestGeminiClientRejectsUnknownAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"candidates":[{"content":{"parts":[{"text":"{\"event_type\":\"X\",\"direction\":\"NEUTRAL\",\"severity\":\"LOW\",\"confidence\":0.5,\"affected_assets\":[\"BTCUSDT\"],\"action\":\"SELL_EVERYTHING\",\"source_count\":1,\"has_official_source\":false}"}]}}]}`
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := gemini.NewClient(server.URL, "test-key", "gemini-2.5-flash", 5*time.Second, 16*1024)

	_, err := client.Analyze(context.Background(), output.NewsAnalysisInput{
		Source: "reuters.com", URL: "https://reuters.com/a", Title: "t", Content: "c", PublishedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for an action outside the deterministic enum (NORMAL/REDUCE/BLOCK_NEW_ENTRIES/REVIEW_REQUIRED/KILL_SWITCH), got nil")
	}
}

func TestGeminiClientRejectsEmptyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer server.Close()

	client := gemini.NewClient(server.URL, "test-key", "gemini-2.5-flash", 5*time.Second, 16*1024)

	_, err := client.Analyze(context.Background(), output.NewsAnalysisInput{
		Source: "reuters.com", URL: "https://reuters.com/a", Title: "t", Content: "c", PublishedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for an empty candidates list, got nil")
	}
}
