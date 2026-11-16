// Command server runs the calculator REST API.
//
// Environment variables:
//
//	PORT            port to listen on (default 8080)
//	ALLOWED_ORIGIN  browser origin allowed by CORS (default http://localhost:5173)
//	STATIC_DIR      directory of built frontend files to serve (default: unset, API only)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Julianmotta07/sezzle-calculator/backend/internal/api"
)

// shutdownTimeout is how long in-flight requests get to finish on shutdown.
const shutdownTimeout = 10 * time.Second

func main() {
	// Cancelled on SIGINT (Ctrl+C) or SIGTERM (docker stop), which triggers a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, nil)
	stop()
	if err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until it fails or ctx is cancelled,
// then shuts down gracefully. If onListen is not nil, it is called with the
// bound address once the server accepts connections (tests use it with PORT=0).
func run(ctx context.Context, onListen func(net.Addr)) error {
	port := envOr("PORT", "8080")
	origin := envOr("ALLOWED_ORIGIN", "http://localhost:5173")
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir != "" {
		// Fail fast on a misconfigured path instead of serving 404s at runtime.
		if _, err := os.Stat(filepath.Join(staticDir, "index.html")); err != nil {
			return fmt.Errorf("STATIC_DIR %q has no index.html: %w", staticDir, err)
		}
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.NewHandler(api.Config{AllowedOrigin: origin, StaticDir: staticDir}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen + Serve is what ListenAndServe does; splitting it reports a busy
	// port immediately and exposes the real address when PORT=0.
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	slog.Info("server listening", "addr", srv.Addr, "allowed_origin", origin, "static_dir", staticDir)
	if onListen != nil {
		onListen(ln.Addr())
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

	select {
	case err := <-serveErr:
		// Serve only returns early on failure.
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down", "timeout", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("server stopped")
	return nil
}

// envOr returns the environment variable key, or fallback if it is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
