// Package connection wraps a single dashboard WebSocket connection: a
// buffered send channel plus the read/write pumps gorilla/websocket
// requires (one goroutine per direction).
package connection

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"trading-core/internal/interfaces/websocket/hub"
)

const (
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	maxMessageBytes = 4096
)

var (
	ErrClosed     = errors.New("connection closed")
	ErrSendBuffer = errors.New("send buffer full")
)

// Connection implements hub.Client.
type Connection struct {
	conn   *websocket.Conn
	send   chan []byte
	logger *slog.Logger

	closeOnce sync.Once
	closed    chan struct{}
}

func New(conn *websocket.Conn, logger *slog.Logger) *Connection {
	return &Connection{
		conn:   conn,
		send:   make(chan []byte, 32),
		logger: logger,
		closed: make(chan struct{}),
	}
}

var _ hub.Client = (*Connection)(nil)

func (c *Connection) Send(message []byte) error {
	select {
	case <-c.closed:
		return ErrClosed
	default:
	}
	select {
	case c.send <- message:
		return nil
	default:
		return ErrSendBuffer
	}
}

func (c *Connection) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		close(c.send)
	})
	return c.conn.Close()
}

// WritePump drains the send channel to the socket and keeps it alive with
// periodic pings. Run it in its own goroutine per connection; it returns
// once the connection closes.
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump discards inbound messages — this WebSocket is broadcast-only —
// but reading keeps the connection's deadline machinery alive and detects
// disconnects. Run it in its own goroutine; it returns once the connection
// closes.
func (c *Connection) ReadPump() {
	c.conn.SetReadLimit(maxMessageBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
