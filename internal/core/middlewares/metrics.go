package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"sentinel/internal/core/monitoring"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		routeContext := chi.RouteContext(r.Context())
		route := "unknown"
		if routeContext != nil && routeContext.RoutePattern() != "" {
			route = routeContext.RoutePattern()
		}

		status := strconv.Itoa(ww.Status())
		duration := time.Since(start).Seconds()

		monitoring.HTTPRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
	})
}
