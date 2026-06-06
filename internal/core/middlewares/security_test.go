package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	recorder := rec.Result()
	defer recorder.Body.Close()

	cases := []struct {
		header   string
		expected string
	}{
		{"Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "no-referrer"},
	}
	for _, c := range cases {
		got := recorder.Header.Get(c.header)
		if got != c.expected {
			t.Errorf("header %q = %q, want %q", c.header, got, c.expected)
		}
	}
}

func TestSecurityHeaders_HeadersPresentOnErrorResponses(t *testing.T) {
	// Even when a downstream handler returns a non-2xx, the headers must
	// still be set so error responses are not weaker than success ones.
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Result().StatusCode; got != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", got, http.StatusInternalServerError)
	}
	if got := rec.Result().Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options on error response = %q, want %q", got, "nosniff")
	}
}
