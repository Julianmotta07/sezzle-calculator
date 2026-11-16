// Package api exposes the calculator over a JSON REST API built on net/http.
package api

import "net/http"

// Config holds the settings the HTTP layer needs.
type Config struct {
	// AllowedOrigin is the single browser origin allowed by CORS,
	// for example "http://localhost:5173". Use "*" to allow any origin.
	AllowedOrigin string

	// StaticDir, when non-empty, is a directory of built frontend files
	// (containing index.html) served on every path not used by the API.
	StaticDir string
}

// NewHandler returns the API's root handler: all routes wrapped in CORS.
func NewHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/v1/operations", handleOperations)
	mux.HandleFunc("POST /api/v1/calculate", handleCalculate)

	// Method-less patterns for the same paths. The mux prefers the
	// method-specific patterns above, so these only catch wrong methods and
	// let us return 405 in the shared JSON error shape (the mux's built-in
	// 405 is plain text).
	mux.Handle("/health", methodNotAllowed("GET, HEAD"))
	mux.Handle("/api/v1/operations", methodNotAllowed("GET, HEAD"))
	mux.Handle("/api/v1/calculate", methodNotAllowed("POST"))

	// Unknown API paths are always a JSON 404, even when serving the frontend.
	mux.HandleFunc("/api/", handleNotFound)
	mux.HandleFunc("/health/", handleNotFound)

	// Everything else is the frontend if configured, otherwise a JSON 404.
	if cfg.StaticDir != "" {
		mux.Handle("/", staticHandler(cfg.StaticDir))
	} else {
		mux.HandleFunc("/", handleNotFound)
	}

	return cors(cfg.AllowedOrigin, mux)
}

// handleNotFound answers unknown paths with a JSON 404.
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, &apiError{http.StatusNotFound, CodeNotFound, "no route for " + r.URL.Path})
}

// methodNotAllowed answers a known path called with the wrong method.
func methodNotAllowed(allow string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, &apiError{http.StatusMethodNotAllowed, CodeMethodNotAllowed,
			"method " + r.Method + " is not allowed; use " + allow})
	})
}

// cors adds CORS headers for allowedOrigin and answers preflight requests.
// Requests from other origins get no CORS headers, so browsers block them.
func cors(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")

		origin := r.Header.Get("Origin")
		allowed := origin != "" && (allowedOrigin == "*" || origin == allowedOrigin)
		if allowed {
			if allowedOrigin == "*" {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
			}
		}

		isPreflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
		if isPreflight {
			if allowed {
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Max-Age", "600")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
