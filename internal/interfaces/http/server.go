// Package http assembles the HTTP server around the router built in
// internal/interfaces/http/router. internal/app owns its lifecycle
// (start/graceful shutdown).
package http

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, handler http.Handler, readTimeout, writeTimeout, idleTimeout time.Duration) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}
}

// Start blocks serving until the listener errors or Shutdown is called
// (which makes it return http.ErrServerClosed — not an error the caller
// needs to report).
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
