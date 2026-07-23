package failure_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trading-core/internal/infrastructure/external/broker/binance"
)

// TestBinanceBrokerTimesOutOnSlowServer proves the broker adapter respects
// a caller-supplied deadline instead of hanging forever when the exchange
// is slow/unresponsive — the scenario ExecuteApprovedOrder's "reconcile
// before retry" logic (section 8.9) exists to handle.
func TestBinanceBrokerTimesOutOnSlowServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalWalletBalance":"1000","totalMarginBalance":"1000","availableBalance":"1000"}`))
	}))
	defer server.Close()

	// The client's own HTTP timeout is longer than the server's delay, but
	// the caller's context deadline is shorter — the request must still be
	// cancelled promptly instead of the caller blocking for the full
	// client timeout.
	client := binance.NewClient(server.URL, "key", "secret", 5000, 10*time.Second)
	broker := binance.NewBroker(client)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := broker.GetAccount(ctx)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got error %v, want one wrapping context.DeadlineExceeded", err)
	}
}
