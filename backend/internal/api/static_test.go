package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Julianmotta07/sezzle-calculator/backend/internal/api"
)

const (
	indexHTML = "<!doctype html><title>Calculator</title>"
	appJS     = "console.log('app')"
	secret    = "top secret"
)

// newStaticHandler builds the API handler serving a fake frontend build.
func newStaticHandler(t *testing.T) http.Handler {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "static")
	// A file just outside the static directory that must never be served.
	writeFile(t, filepath.Join(base, "secret.txt"), secret)
	writeFile(t, filepath.Join(dir, "index.html"), indexHTML)
	writeFile(t, filepath.Join(dir, "favicon.svg"), "<svg/>")
	writeFile(t, filepath.Join(dir, "assets", "index-abc123.js"), appJS)
	return api.NewHandler(api.Config{AllowedOrigin: testOrigin, StaticDir: dir})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStaticFiles(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		wantBody  string
		wantCache string
	}{
		{"root serves index.html", "/", indexHTML, "no-cache"},
		{"hashed asset is cached long-term", "/assets/index-abc123.js", appJS, "public, max-age=31536000, immutable"},
		{"other file is revalidated", "/favicon.svg", "<svg/>", "no-cache"},
		{"unknown route falls back to index.html", "/history/42", indexHTML, "no-cache"},
		{"missing asset falls back to index.html", "/assets/missing.js", indexHTML, "no-cache"},
		{"directory is not listed", "/assets/", indexHTML, "no-cache"},
		{"file outside the directory is not served", "/secret.txt", indexHTML, "no-cache"},
		{"encoded traversal stays inside the directory", "/assets/%2e%2e/%2e%2e/secret.txt", indexHTML, "no-cache"},
	}
	h := newStaticHandler(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			if got := rec.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
			if got := rec.Header().Get("Cache-Control"); got != tc.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCache)
			}
		})
	}
}

func TestStaticTraversalIsRedirectedInside(t *testing.T) {
	// The mux cleans "/../secret.txt" and redirects to "/secret.txt",
	// which is resolved inside the static directory (see TestStaticFiles).
	rec := do(t, newStaticHandler(t), http.MethodGet, "/../secret.txt", "")
	if strings.Contains(rec.Body.String(), secret) {
		t.Fatal("served a file outside the static directory")
	}
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/secret.txt" {
		t.Errorf("got %d Location=%q, want 301 to /secret.txt", rec.Code, rec.Header().Get("Location"))
	}
}

func TestStaticHead(t *testing.T) {
	rec := do(t, newStaticHandler(t), http.MethodHead, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD response has a body: %q", rec.Body.String())
	}
}

func TestStaticKeepsAPIRoutes(t *testing.T) {
	h := newStaticHandler(t)

	t.Run("health", func(t *testing.T) {
		rec := do(t, h, http.MethodGet, "/health", "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Errorf("got %d %s, want 200 status ok", rec.Code, rec.Body.String())
		}
	})

	t.Run("calculate", func(t *testing.T) {
		rec := do(t, h, http.MethodPost, "/api/v1/calculate", `{"operation":"add","a":2,"b":3}`)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"result":5`) {
			t.Errorf("got %d %s, want 200 result 5", rec.Code, rec.Body.String())
		}
	})

	errorCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{"unknown API path is a JSON 404, not index.html", http.MethodGet, "/api/v1/nope", 404, api.CodeNotFound},
		{"unknown API root path", http.MethodGet, "/api/", 404, api.CodeNotFound},
		{"health subpath", http.MethodGet, "/health/extra", 404, api.CodeNotFound},
		{"wrong method on API route", http.MethodGet, "/api/v1/calculate", 405, api.CodeMethodNotAllowed},
		{"wrong method on static route", http.MethodPost, "/", 405, api.CodeMethodNotAllowed},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, "")
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestStaticDirWithoutIndexIsJSON404(t *testing.T) {
	h := api.NewHandler(api.Config{AllowedOrigin: testOrigin, StaticDir: t.TempDir()})
	rec := do(t, h, http.MethodGet, "/anything", "")
	assertError(t, rec, http.StatusNotFound, api.CodeNotFound)
}

func TestNoStaticDirKeepsJSON404(t *testing.T) {
	rec := do(t, newHandler(), http.MethodGet, "/", "")
	assertError(t, rec, http.StatusNotFound, api.CodeNotFound)
}
