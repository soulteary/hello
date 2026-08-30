// Package httpserver provides the optional HTTP mode used when hello acts as
// a small protected-service backend behind a reverse proxy.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand/v2"
	"mime"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/soulteary/hello/internal/animation"
)

const (
	shutdownTimeout    = 5 * time.Second
	streamWriteTimeout = 5 * time.Second
	defaultFrameDelay  = 75 * time.Millisecond
	maxFrameDelay      = 60 * time.Second
	defaultAnimation   = "parrot"
)

var browserColors = []string{
	"#ff8787",
	"#ffd166",
	"#73e2a7",
	"#67d5e8",
	"#82aaff",
	"#c792ea",
	"#f5a9cb",
	"#ff79c6",
	"#ff6b9a",
	"#ff6b6b",
}

// Options configures the HTTP server.
type Options struct {
	Addr       string
	Version    string
	Animation  string
	FrameDelay time.Duration
	Mono       bool
	Stdout     io.Writer
}

// HandlerOptions configures the HTTP handler independently of its listener.
// Inventory is primarily useful to callers that need a custom embedded
// catalog; a nil inventory loads the animations shipped with hello.
type HandlerOptions struct {
	Version    string
	Animation  string
	FrameDelay time.Duration
	Mono       bool
	Inventory  animation.Inventory
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

	handler, err := NewHandlerWithOptions(HandlerOptions{
		Version:    opts.Version,
		Animation:  opts.Animation,
		FrameDelay: opts.FrameDelay,
		Mono:       opts.Mono,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	requestCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	server := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return requestCtx },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// Streaming responses cannot use a server-wide WriteTimeout. The SSE
		// handler applies a deadline to every individual write instead.
		WriteTimeout:   0,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 64 << 10,
	}
	server.RegisterOnShutdown(cancelRequests)
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

// NewHandler returns a handler with the bundled parrot and default frame
// delay. It is retained as a compact entry point for tests and embedders.
func NewHandler(version string) http.Handler {
	handler, err := NewHandlerWithOptions(HandlerOptions{Version: version})
	if err != nil {
		panic(err)
	}
	return handler
}

// NewHandlerWithOptions builds the content-negotiating root handler, health
// endpoint and browser event stream.
func NewHandlerWithOptions(opts HandlerOptions) (http.Handler, error) {
	if opts.FrameDelay == 0 {
		opts.FrameDelay = defaultFrameDelay
	}
	if opts.FrameDelay < time.Millisecond || opts.FrameDelay > maxFrameDelay {
		return nil, fmt.Errorf("frame delay must be between 1ms and %s", maxFrameDelay)
	}
	if opts.Inventory == nil {
		opts.Inventory = animation.NewInventory()
	}

	name := strings.TrimSpace(opts.Animation)
	if name == "" {
		name = defaultAnimation
	}
	anim, ok := opts.Inventory[name]
	if !ok {
		return nil, fmt.Errorf("animation %q not found", name)
	}
	if len(anim.Frames) == 0 {
		return nil, fmt.Errorf("animation %q has no frames", name)
	}

	app := handler{
		version:    strings.TrimSpace(opts.Version),
		name:       name,
		animation:  anim,
		textFrames: cacheDistinctFrames(anim.Frames),
		frameDelay: opts.FrameDelay,
		mono:       opts.Mono,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.health)
	mux.HandleFunc("/events", app.events)
	mux.HandleFunc("/", app.root)
	return mux, nil
}

type handler struct {
	version    string
	name       string
	animation  animation.Animation
	textFrames []string
	frameDelay time.Duration
	mono       bool
}

func (h handler) health(w http.ResponseWriter, r *http.Request) {
	setResponseHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, "ok\n")
	}
}

func (h handler) root(w http.ResponseWriter, r *http.Request) {
	setResponseHeaders(w)
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}

	format, err := negotiateResponseFormat(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if format == formatHTML {
		h.html(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeTextResponse(w, r, h.version, randomFrame(h.textFrames))
}

func (h handler) html(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_ = browserPage.Execute(w, struct {
		Version   string
		Animation string
		Frame     string
		Mono      bool
	}{
		Version:   h.version,
		Animation: h.name,
		Frame:     string(h.animation.Frames[0]),
		Mono:      h.mono,
	})
}

func (h handler) events(w http.ResponseWriter, r *http.Request) {
	setResponseHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	controller := http.NewResponseController(w)
	prepare := func() error {
		err := controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout))
		if errors.Is(err, http.ErrNotSupported) {
			return nil
		}
		return err
	}
	if err := prepare(); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	// ResponseController follows Unwrap methods exposed by middleware. Probe
	// through it rather than requiring the outermost writer to implement
	// http.Flusher directly.
	if err := controller.Flush(); err != nil {
		if errors.Is(err, http.ErrNotSupported) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		}
		return
	}
	_ = streamAnimation(r.Context(), w, prepare, controller.Flush, h.name, h.animation, h.frameDelay, h.mono)
}

func setResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

type responseFormat uint8

const (
	formatText responseFormat = iota
	formatHTML
)

func negotiateResponseFormat(r *http.Request) (responseFormat, error) {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) {
	case "html":
		return formatHTML, nil
	case "text", "plain":
		return formatText, nil
	case "":
		// Continue with request-header negotiation.
	default:
		return formatText, errors.New("format must be html or text")
	}

	userAgent := strings.ToLower(r.UserAgent())
	if strings.Contains(userAgent, "curl/") {
		return formatText, nil
	}
	if acceptsHTML(r.Header.Get("Accept")) || strings.Contains(userAgent, "mozilla/") {
		return formatHTML, nil
	}
	return formatText, nil
}

func acceptsHTML(accept string) bool {
	for _, value := range strings.Split(accept, ",") {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
			continue
		}
		quality := 1.0
		if raw := params["q"]; raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}
			quality = parsed
		}
		if quality > 0 {
			return true
		}
	}
	return false
}

// cacheDistinctFrames detaches immutable text copies from the animation
// buffers and removes duplicates so every visual frame has equal weight.
func cacheDistinctFrames(frames [][]byte) []string {
	cached := make([]string, 0, len(frames))
	seen := make(map[string]struct{}, len(frames))
	for _, frame := range frames {
		text := string(frame)
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		cached = append(cached, text)
	}
	return cached
}

func randomFrame(frames []string) string {
	if len(frames) == 0 {
		return ""
	}
	// The package-level generator is safe for concurrent request handlers.
	return frames[rand.IntN(len(frames))]
}

func writeTextResponse(w io.Writer, r *http.Request, version, frame string) {
	_, _ = io.WriteString(w, frame)
	if frame == "" || frame[len(frame)-1] != '\n' {
		_, _ = io.WriteString(w, "\n")
	}
	_, _ = io.WriteString(w, "\n")
	writeRequestSummary(w, r, version)
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
	switch http.CanonicalHeaderKey(name) {
	case "Forwarded",
		"User-Agent",
		"X-Real-Ip",
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"X-Forwarded-Method",
		"X-Forwarded-Port",
		"X-Forwarded-Prefix",
		"X-Forwarded-Proto",
		"X-Forwarded-Uri",
		"X-Forwarded-User",
		"X-Auth-Email",
		"X-Auth-Groups",
		"X-Auth-Name",
		"X-Auth-Role",
		"X-Auth-Scopes",
		"X-Auth-User":
		return true
	default:
		return false
	}
}

type frameEvent struct {
	Animation string `json:"animation"`
	Color     string `json:"color"`
	Frame     string `json:"frame"`
	Index     int    `json:"index"`
}

func streamAnimation(
	ctx context.Context,
	w io.Writer,
	prepare func() error,
	flush func() error,
	name string,
	anim animation.Animation,
	delay time.Duration,
	mono bool,
) error {
	if len(anim.Frames) == 0 {
		return errors.New("animation has no frames")
	}
	if delay <= 0 {
		return errors.New("frame delay must be positive")
	}

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	var sequence uint64
	frameIndex := 0
	colorIndex := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if prepare != nil {
			if err := prepare(); err != nil {
				return err
			}
		}
		color := "#d6e2f0"
		if !mono {
			color = browserColors[colorIndex%len(browserColors)]
		}
		if err := writeFrameEvent(w, sequence, frameEvent{
			Animation: name,
			Color:     color,
			Frame:     string(anim.Frames[frameIndex]),
			Index:     frameIndex,
		}); err != nil {
			return err
		}
		if flush != nil {
			if err := flush(); err != nil {
				return err
			}
		}

		sequence++
		frameIndex = (frameIndex + 1) % len(anim.Frames)
		if !mono {
			colorIndex = (colorIndex + 1) % len(browserColors)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func writeFrameEvent(w io.Writer, sequence uint64, event frameEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "id: %d\nevent: frame\nretry: 1000\ndata: %s\n\n", sequence, payload)
	return err
}

var browserPage = template.Must(template.New("browser").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>hello · {{.Animation}}</title>
  <style>
    :root { color-scheme: dark; font-family: Inter, ui-sans-serif, system-ui, sans-serif; }
    * { box-sizing: border-box; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; padding: 24px; background: radial-gradient(circle at top, #1d2a3b, #090d13 65%); color: #d6e2f0; }
    main { width: min(960px, 100%); }
    h1 { margin: 0 0 8px; font-size: clamp(1.5rem, 4vw, 2.5rem); letter-spacing: -.03em; }
    .lede { margin: 0 0 20px; color: #90a4bb; }
    .terminal { overflow: hidden; border: 1px solid #2b3a4c; border-radius: 14px; background: rgba(5, 9, 14, .94); box-shadow: 0 24px 80px rgba(0, 0, 0, .45); }
    .bar { display: flex; align-items: center; gap: 7px; min-height: 42px; padding: 0 14px; border-bottom: 1px solid #253244; background: #111923; color: #7f93aa; font: 12px ui-monospace, SFMono-Regular, Consolas, monospace; }
    .dot { width: 10px; height: 10px; border-radius: 50%; background: #ff6b6b; box-shadow: 17px 0 #ffd166, 34px 0 #73e2a7; margin-right: 34px; }
    pre { min-height: 24rem; margin: 0; overflow: auto; padding: clamp(16px, 4vw, 32px); color: {{if .Mono}}#d6e2f0{{else}}#ff8787{{end}}; font: clamp(8px, 1.35vw, 16px)/1.08 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace; white-space: pre; }
    .status { margin: 12px 2px 0; color: #71859d; font-size: 13px; }
    code { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
  </style>
</head>
<body>
  <main>
    <h1>hello from the terminal</h1>
    <p class="lede">The server is pushing the <code>{{.Animation}}</code> animation one ASCII frame at a time.</p>
    <section class="terminal" aria-label="Animated ASCII art terminal">
      <div class="bar"><span class="dot" aria-hidden="true"></span>hello {{if .Version}}· {{.Version}}{{end}}</div>
      <pre id="screen" aria-live="off">{{.Frame}}</pre>
    </section>
    <p id="status" class="status" role="status">Connecting to the event stream…</p>
    <noscript><p class="status">JavaScript is disabled, so this page shows the first frame only.</p></noscript>
  </main>
  <script>
    (() => {
      const screen = document.getElementById("screen");
      const status = document.getElementById("status");
      if (!("EventSource" in window)) {
        status.textContent = "This browser does not support server-sent events; showing a static frame.";
        return;
      }
      const eventsURL = new URL(window.location.href);
      eventsURL.search = "";
      eventsURL.hash = "";
      eventsURL.pathname = eventsURL.pathname.replace(/\/?$/, "/events");
      const source = new EventSource(eventsURL);
      source.addEventListener("open", () => { status.textContent = "Live · server-sent events"; });
      source.addEventListener("frame", (event) => {
        try {
          const payload = JSON.parse(event.data);
          screen.textContent = payload.frame;
          screen.style.color = payload.color;
        } catch (_) {
          status.textContent = "Received an invalid frame; reconnecting…";
        }
      });
      source.addEventListener("error", () => { status.textContent = "Stream interrupted · reconnecting automatically…"; });
    })();
  </script>
</body>
</html>`))
