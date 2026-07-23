package middleware

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// Timeout bounds how long a handler may run before the client receives a
// 503. It wraps chi's implementation for consistency with the rest of the
// chain, which is otherwise hand-written.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return chimw.Timeout(d)
}
