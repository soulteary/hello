// Package httpserver provides the optional HTTP mode used when hello acts as
// a small protected-service backend behind a reverse proxy.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const shutdownTimeout = 5 * time.Second

// Options configures the HTTP server.
type Options struct {
	Addr    string
	Version string
	Stdout  io.Writer
}

// Run serves HTTP until ctx is cancelled or the listener fails.
func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	addr := strings.TrimSpace(opts.Addr)
	if addr == "" {
		return errors.New("listen address is required")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	server := &http.Server{
		Handler:           NewHandler(opts.Version),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	fmt.Fprintf(opts.Stdout, "hello HTTP server listening on %s\n", listener.Addr())
	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		err := <-serveErr
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// NewHandler returns the HTTP-mode handler. The root response is deliberately
// small and only reflects reverse-proxy identity/routing headers. Credentials
// such as Cookie and Authorization are never copied into the response.
func NewHandler(version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		setResponseHeaders(w)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, "ok\n")
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		setResponseHeaders(w)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeRequestSummary(w, r, version)
	})
	return mux
}

func setResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeRequestSummary(w io.Writer, r *http.Request, version string) {
	hostname, _ := os.Hostname()
	fmt.Fprintln(w, "Hello from soulteary/hello!")
	if strings.TrimSpace(version) != "" {
		fmt.Fprintf(w, "Version: %s\n", version)
	}
	if hostname != "" {
		fmt.Fprintf(w, "Hostname: %s\n", hostname)
	}
	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "URL: %s\n", r.URL.RequestURI())
	fmt.Fprintf(w, "Host: %s\n", r.Host)

	keys := make([]string, 0, len(r.Header))
	for key := range r.Header {
		if safeReflectedHeader(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range r.Header.Values(key) {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}
}

func safeReflectedHeader(name string) bool {
	name = http.CanonicalHeaderKey(name)
	return name == "User-Agent" || name == "X-Real-Ip" ||
		strings.HasPrefix(name, "X-Forwarded-") ||
		strings.HasPrefix(name, "X-Auth-")
}
