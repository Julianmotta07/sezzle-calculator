package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests use t.Setenv, so they must not run in parallel.

func TestEnvOr(t *testing.T) {
	const key = "CALC_TEST_ENV_OR"
	cases := []struct {
		name  string
		value *string // nil = unset
		want  string
	}{
		{"unset uses fallback", nil, "fallback"},
		{"empty uses fallback", ptr(""), "fallback"},
		{"set value wins", ptr("9090"), "9090"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(key, "") // registers cleanup that restores the original value
			if tc.value == nil {
				os.Unsetenv(key)
			} else {
				t.Setenv(key, *tc.value)
			}
			if got := envOr(key, "fallback"); got != tc.want {
				t.Errorf("envOr = %q, want %q", got, tc.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// startRun runs run() in the background on a random free port and returns
// the base URL, a cancel func (simulates SIGINT/SIGTERM) and run's result channel.
func startRun(t *testing.T) (baseURL string, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	t.Setenv("PORT", "0")

	ctx, cancel := context.WithCancel(context.Background())
	addrCh := make(chan net.Addr, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, func(a net.Addr) { addrCh <- a })
	}()

	select {
	case a := <-addrCh:
		t.Cleanup(cancel)
		return "http://" + a.String(), cancel, errCh
	case err := <-errCh:
		cancel()
		t.Fatalf("run returned before listening: %v", err)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("server did not start listening within 5s")
	}
	return "", nil, nil
}

// waitStopped asserts that run returns nil promptly after cancellation.
func waitStopped(t *testing.T, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v after shutdown, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return within 5s of cancellation")
	}
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp.StatusCode, string(body)
}

func TestRunServesAPIAndShutsDownGracefully(t *testing.T) {
	t.Setenv("STATIC_DIR", "")
	baseURL, cancel, done := startRun(t)

	status, body := get(t, baseURL+"/health")
	if status != http.StatusOK || !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("GET /health = %d %s, want 200 status ok", status, body)
	}

	waitStopped(t, cancel, done)

	if _, err := http.Get(baseURL + "/health"); err == nil {
		t.Error("server still accepting connections after shutdown")
	}
}

func TestRunServesStaticDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<title>Calculator</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STATIC_DIR", dir)
	baseURL, cancel, done := startRun(t)

	status, body := get(t, baseURL+"/")
	if status != http.StatusOK || !strings.Contains(body, "<title>Calculator</title>") {
		t.Errorf("GET / = %d %s, want 200 with index.html", status, body)
	}

	waitStopped(t, cancel, done)
}

func TestRunRejectsStaticDirWithoutIndex(t *testing.T) {
	t.Setenv("PORT", "0")
	t.Setenv("STATIC_DIR", t.TempDir())

	err := run(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "has no index.html") {
		t.Fatalf("run error = %v, want STATIC_DIR has no index.html", err)
	}
}

func TestRunFailsWhenPortIsInUse(t *testing.T) {
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	_, port, _ := net.SplitHostPort(busy.Addr().String())
	t.Setenv("PORT", port)
	t.Setenv("STATIC_DIR", "")

	if err := run(context.Background(), nil); err == nil {
		t.Fatal("run succeeded on a port that is already in use")
	}
}
