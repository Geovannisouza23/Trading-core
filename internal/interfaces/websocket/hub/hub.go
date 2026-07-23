// Package hub fans domain events out to every connected dashboard
// WebSocket client. This is the public dashboard WebSocket — not to be
// confused with internal/infrastructure/external/marketdata/websocket,
// which is the exchange-facing client.
package hub

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Message is the envelope broadcast to every connected client.
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Client is the minimal surface the hub needs, so it never has to know
// about gorilla/websocket directly.
type Client interface {
	Send(message []byte) error
	Close() error
}

type Hub struct {
	mu      sync.RWMutex
	clients map[Client]struct{}
	logger  *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{clients: make(map[Client]struct{}), logger: logger}
}

func (h *Hub) Register(c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) Unregister(c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	_ = c.Close()
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(msgType string, payload any) {
	body, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		h.logger.Error("websocket: failed to marshal broadcast message", "type", msgType, "error", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if err := c.Send(body); err != nil {
			h.logger.Warn("websocket: dropping slow/closed client", "error", err)
		}
	}
}
