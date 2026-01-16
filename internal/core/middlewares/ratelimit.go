package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimit Example: RateLimit(100, 1*time.Minute) -> 100 requests per minute per IP.
func RateLimit(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.Limit(
		requestLimit,
		window,
		httprate.WithKeyFuncs(httprate.KeyByIP), // Limit by IP Address
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"status":"error", "message": "Too many requests. Please slow down."}`))
		}),
	)
}
