package middlewares

import "net/http"

// SecurityHeaders sets baseline HTTP security headers on every response.
//
// These are scoped to what a JSON-only API actually needs; the values are
// intentionally strict because the API never serves HTML, scripts, or frames.
//
// - Content-Security-Policy: deny everything by default. The API serves only
//   application/json, so no inline/eval/frame/connect-src should be permitted
//   from a browser context.
// - X-Content-Type-Options: prevents MIME-sniffing-driven content-type changes.
// - Referrer-Policy: never leak the request URL (which can include resource IDs)
//   to upstream services via the Referer header.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
