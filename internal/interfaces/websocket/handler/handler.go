// Package handler upgrades an incoming HTTP request to a dashboard
// WebSocket connection and registers it with the hub.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"trading-core/internal/interfaces/websocket/connection"
	"trading-core/internal/interfaces/websocket/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// The dashboard WebSocket carries no cookies/session state — auth for
	// the MVP dashboard feed is out of scope, unlike the REST API which is
	// JWT-gated. CORS on the REST API still governs which origins can load
	// the dashboard app that opens this connection.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub    *hub.Hub
	logger *slog.Logger
}

func NewHandler(h *hub.Hub, logger *slog.Logger) *Handler {
	return &Handler{hub: h, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket: upgrade failed", "error", err)
		return
	}

	c := connection.New(conn, h.logger)
	h.hub.Register(c)

	go func() {
		c.ReadPump()
		h.hub.Unregister(c)
	}()
	go c.WritePump()
}
