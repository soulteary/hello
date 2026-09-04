package httpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soulteary/hello/internal/animation"
)

func smallInventory() animation.Inventory {
	return animation.Inventory{
		"parrot": {
			Metadata: map[string]string{"description": "test parrot"},
			Frames:   [][]byte{[]byte("PARROT-A"), []byte("PARROT-B\n")},
		},
		"cat": {
			Metadata: map[string]string{"description": "test cat"},
			Frames:   [][]byte{[]byte("CAT-A"), []byte("CAT-B")},
		},
	}
}

func newSmallHandler(t *testing.T, opts HandlerOptions) http.Handler {
	t.Helper()
	if opts.Inventory == nil {
		opts.Inventory = smallInventory()
	}
	handler, err := NewHandlerWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestRootContentNegotiation(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{Version: "2.0.0"})
	tests := []struct {
		name        string
		url         string
		userAgent   string
		accept      string
		contentType string
		contains    []string
		textFrames  []string
	}{
		{
			name:        "curl receives one plain frame and diagnostics",
			url:         "http://hello.example/",
			userAgent:   "curl/8.5.0",
			accept:      "*/*",
			contentType: "text/plain",
			contains:    []string{"Hello from soulteary/hello!\n\nVersion: 2.0.0", "Project: " + projectURL},
			textFrames:  []string{"PARROT-A", "PARROT-B\n"},
		},
		{
			name:        "browser accept receives HTML",
			url:         "http://hello.example/",
			userAgent:   "test-browser",
			accept:      "text/html,application/xhtml+xml",
			contentType: "text/html",
			contains:    []string{"<!doctype html>", "PARROT-A", "new EventSource(eventsURL)", `Project: <a href="` + projectURL + `">` + projectURL + `</a>`},
		},
		{
			name:        "browser user agent fallback",
			url:         "http://hello.example/",
			userAgent:   "Mozilla/5.0",
			contentType: "text/html",
			contains:    []string{"server-sent events"},
		},
		{
			name:        "explicit HTML overrides curl",
			url:         "http://hello.example/?format=html",
			userAgent:   "curl/8.5.0",
			contentType: "text/html",
			contains:    []string{"<title>hello"},
		},
		{
			name:        "explicit text overrides browser",
			url:         "http://hello.example/?format=plain",
			userAgent:   "Mozilla/5.0",
			accept:      "text/html",
			contentType: "text/plain",
			textFrames:  []string{"PARROT-A", "PARROT-B\n"},
		},
		{
			name:        "generic client defaults to text",
			url:         "http://hello.example/",
			userAgent:   "hello-test",
			accept:      "*/*",
			contentType: "text/plain",
			textFrames:  []string{"PARROT-A", "PARROT-B\n"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			req.Header.Set("User-Agent", tc.userAgent)
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			response := recorder.Result()
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d; body: %s", response.StatusCode, body)
			}
			if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, tc.contentType) {
				t.Errorf("Content-Type = %q, want prefix %q", got, tc.contentType)
			}
			if got := response.Header.Get(projectHeader); got != projectURL {
				t.Errorf("%s header = %q, want %q", projectHeader, got, projectURL)
			}
			for _, want := range tc.contains {
				if !bytes.Contains(body, []byte(want)) {
					t.Errorf("response does not contain %q: %s", want, body)
				}
			}
			if len(tc.textFrames) > 0 && !hasTextFramePrefix(body, tc.textFrames) {
				t.Errorf("response does not start with a cached frame: %s", body)
			}
			for _, name := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Permissions-Policy"} {
				if response.Header.Get(name) == "" {
					t.Errorf("security header %s is missing", name)
				}
			}
		})
	}
}

func TestRootHeadIncludesProjectHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "http://hello.example/", nil)
	req.Header.Set("User-Agent", "curl/8.5.0")
	recorder := httptest.NewRecorder()

	newSmallHandler(t, HandlerOptions{}).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("HEAD response has a body: %q", recorder.Body.String())
	}
	if got := recorder.Header().Get(projectHeader); got != projectURL {
		t.Errorf("%s header = %q, want %q", projectHeader, got, projectURL)
	}
}

func TestTextFrameCacheIsDistinctAndImmutable(t *testing.T) {
	frames := [][]byte{[]byte("FRAME-A"), []byte("FRAME-B\n"), []byte("FRAME-A")}
	cached := cacheDistinctFrames(frames)
	if len(cached) != 2 || cached[0] != "FRAME-A" || cached[1] != "FRAME-B\n" {
		t.Fatalf("cached frames = %#v, want two distinct frames in source order", cached)
	}

	frames[0][0] = 'X'
	if cached[0] != "FRAME-A" {
		t.Fatalf("cached frame changed with source buffer: %q", cached[0])
	}

	for range 64 {
		got := randomFrame(cached)
		if got != "FRAME-A" && got != "FRAME-B\n" {
			t.Fatalf("randomFrame returned uncached frame %q", got)
		}
	}
	if got := randomFrame(nil); got != "" {
		t.Fatalf("randomFrame(nil) = %q, want empty string", got)
	}
}

func BenchmarkRandomCachedFrame(b *testing.B) {
	cached := cacheDistinctFrames(smallInventory()["parrot"].Frames)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = randomFrame(cached)
	}
}

func TestRootRedactsQueryAndIdentityByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/?token=must-not-leak-query", nil)
	req.Header.Set("User-Agent", "curl/8.5.0")
	req.Header.Set("X-Forwarded-User", "test-admin-001")
	req.Header.Set("X-Auth-Email", "admin@example.com")
	req.Header.Set("X-Auth-Token", "must-not-leak-token")
	req.Header.Set("X-Auth-Request-Access-Token", "must-not-leak-auth-request-token")
	req.Header.Set("X-Forwarded-Access-Token", "must-not-leak-forwarded-token")
	req.Header.Set("X-Forwarded-Authorization", "must-not-leak-forwarded-auth")
	req.Header.Set("Authorization", "Bearer must-not-leak-auth")
	req.Header.Set("Cookie", "session=must-not-leak-cookie")
	recorder := httptest.NewRecorder()

	newSmallHandler(t, HandlerOptions{Version: "2.0.0"}).ServeHTTP(recorder, req)
	text := recorder.Body.String()
	for _, want := range []string{"User-Agent: curl/8.5.0", "URL: /"} {
		if !strings.Contains(text, want) {
			t.Errorf("response does not contain %q: %s", want, text)
		}
	}
	for _, secret := range []string{"must-not-leak", "admin@example.com", "test-admin-001", "Authorization:", "Cookie:", "X-Auth-Token:", "X-Forwarded-Access-Token:"} {
		if strings.Contains(text, secret) {
			t.Errorf("response leaked %q: %s", secret, text)
		}
	}

	optIn := httptest.NewRecorder()
	newSmallHandler(t, HandlerOptions{ReflectQuery: true, ReflectIdentity: true}).ServeHTTP(optIn, req)
	for _, want := range []string{
		"URL: /?token=must-not-leak-query",
		"X-Forwarded-User: test-admin-001",
		"X-Auth-Email: admin@example.com",
	} {
		if !strings.Contains(optIn.Body.String(), want) {
			t.Errorf("opt-in response does not contain %q: %s", want, optIn.Body.String())
		}
	}
}

func TestHTMLTemplateEscapesVersion(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{Version: `<script>alert("x")</script>`})
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/?format=html", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	body := recorder.Body.String()
	if strings.Contains(body, `<script>alert("x")</script>`) {
		t.Fatalf("version was not escaped: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("escaped version is missing: %s", body)
	}
	if strings.Contains(body, "ZgotmplZ") {
		t.Errorf("template produced an unsafe-value placeholder: %s", body)
	}
	if !strings.Contains(recorder.Header().Get("Content-Security-Policy"), "connect-src 'self'") {
		t.Errorf("unexpected CSP: %s", recorder.Header().Get("Content-Security-Policy"))
	}
	if strings.Contains(recorder.Header().Get("Content-Security-Policy"), "'unsafe-inline'") {
		t.Errorf("CSP still allows unsafe inline content: %s", recorder.Header().Get("Content-Security-Policy"))
	}
	if !strings.Contains(body, `nonce="`) || !strings.Contains(body, "Pause animation") || !strings.Contains(body, "prefers-reduced-motion") {
		t.Errorf("nonce or accessible animation controls are missing: %s", body)
	}
	wantLink := `Project: <a href="` + projectURL + `">` + projectURL + `</a>`
	if !strings.Contains(body, wantLink) {
		t.Errorf("project footer is missing: %s", body)
	}
}

func TestCustomAnimationAndMonoPage(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{Animation: "cat", Mono: true})

	textRequest := httptest.NewRequest(http.MethodGet, "http://hello.example/?format=text", nil)
	textRecorder := httptest.NewRecorder()
	handler.ServeHTTP(textRecorder, textRequest)
	if !hasTextFramePrefix(textRecorder.Body.Bytes(), []string{"CAT-A", "CAT-B"}) || strings.Contains(textRecorder.Body.String(), "PARROT-A") {
		t.Errorf("custom text animation was not selected: %s", textRecorder.Body.String())
	}

	htmlRequest := httptest.NewRequest(http.MethodGet, "http://hello.example/?format=html", nil)
	htmlRecorder := httptest.NewRecorder()
	handler.ServeHTTP(htmlRecorder, htmlRequest)
	body := htmlRecorder.Body.String()
	for _, want := range []string{"CAT-A", "<code>cat</code>", "color: #d6e2f0"} {
		if !strings.Contains(body, want) {
			t.Errorf("mono custom page does not contain %q: %s", want, body)
		}
	}
}

func TestHealthHeadMethodAndPathContracts(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{})
	tests := []struct {
		name        string
		method      string
		path        string
		accept      string
		status      int
		allow       string
		emptyBody   bool
		contentType string
	}{
		{name: "health get", method: http.MethodGet, path: "/healthz", status: http.StatusOK, contentType: "text/plain"},
		{name: "health head", method: http.MethodHead, path: "/healthz", status: http.StatusOK, emptyBody: true},
		{name: "health post", method: http.MethodPost, path: "/healthz", status: http.StatusMethodNotAllowed, allow: "GET, HEAD"},
		{name: "root text head", method: http.MethodHead, path: "/?format=text", status: http.StatusOK, emptyBody: true, contentType: "text/plain"},
		{name: "root HTML head", method: http.MethodHead, path: "/", accept: "text/html", status: http.StatusOK, emptyBody: true, contentType: "text/html"},
		{name: "root post", method: http.MethodPost, path: "/", status: http.StatusMethodNotAllowed, allow: "GET, HEAD"},
		{name: "events head", method: http.MethodHead, path: "/events", status: http.StatusOK, emptyBody: true, contentType: "text/event-stream"},
		{name: "events post", method: http.MethodPost, path: "/events", status: http.StatusMethodNotAllowed, allow: "GET, HEAD"},
		{name: "missing", method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{name: "invalid format", method: http.MethodGet, path: "/?format=json", status: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://hello.example"+tc.path, nil)
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tc.status, recorder.Body.String())
			}
			if tc.allow != "" && recorder.Header().Get("Allow") != tc.allow {
				t.Errorf("Allow = %q, want %q", recorder.Header().Get("Allow"), tc.allow)
			}
			if tc.emptyBody && recorder.Body.Len() != 0 {
				t.Errorf("HEAD response has a body: %q", recorder.Body.String())
			}
			if tc.contentType != "" && !strings.HasPrefix(recorder.Header().Get("Content-Type"), tc.contentType) {
				t.Errorf("Content-Type = %q, want %q", recorder.Header().Get("Content-Type"), tc.contentType)
			}
		})
	}
}

func TestAcceptsHTML(t *testing.T) {
	tests := map[string]bool{
		"":                                     false,
		"*/*":                                  false,
		"text/plain, text/html":                true,
		"text/html; q=0":                       false,
		"text/html; q=0.2":                     true,
		"application/xhtml+xml":                true,
		"text/html; q=invalid":                 false,
		"broken media type; q=0.5, text/plain": false,
	}
	for header, want := range tests {
		if got := acceptsHTML(header); got != want {
			t.Errorf("acceptsHTML(%q) = %t, want %t", header, got, want)
		}
	}
}

func TestSafeReflectedHeader(t *testing.T) {
	tests := map[string]bool{
		"Forwarded":                       true,
		"X-Forwarded-User":                false,
		"X-Forwarded-For":                 true,
		"X-Forwarded-Proto":               true,
		"X-Auth-User":                     false,
		"X-Auth-Email":                    false,
		"X-Auth-Groups":                   false,
		"X-Auth-Name":                     false,
		"X-Auth-Role":                     false,
		"X-Auth-Scopes":                   false,
		"X-Real-IP":                       true,
		"User-Agent":                      true,
		"Authorization":                   false,
		"Cookie":                          false,
		"Proxy-Authorization":             false,
		"X-Auth-Token":                    false,
		"X-Auth-Request-Access-Token":     false,
		"X-Forwarded-Access-Token":        false,
		"X-Forwarded-Authorization":       false,
		"X-Auth-Unrecognized-Information": false,
		"X-Random":                        false,
	}
	for name, want := range tests {
		if got := safeReflectedHeader(name, false); got != want {
			t.Errorf("safeReflectedHeader(%q, false) = %t, want %t", name, got, want)
		}
	}
	for _, name := range []string{"X-Forwarded-User", "X-Auth-User", "X-Auth-Email", "X-Auth-Groups", "X-Auth-Name", "X-Auth-Role", "X-Auth-Scopes"} {
		if !safeReflectedHeader(name, true) {
			t.Errorf("safeReflectedHeader(%q, true) = false, want true", name)
		}
	}
}

func TestNewHandlerValidation(t *testing.T) {
	tests := []struct {
		name string
		opts HandlerOptions
		want string
	}{
		{name: "negative delay", opts: HandlerOptions{FrameDelay: -time.Millisecond, Inventory: smallInventory()}, want: "frame delay"},
		{name: "sub-millisecond delay", opts: HandlerOptions{FrameDelay: time.Microsecond, Inventory: smallInventory()}, want: "frame delay"},
		{name: "excessive delay", opts: HandlerOptions{FrameDelay: maxFrameDelay + time.Millisecond, Inventory: smallInventory()}, want: "frame delay"},
		{name: "negative stream limit", opts: HandlerOptions{MaxStreams: -1, Inventory: smallInventory()}, want: "maximum event streams"},
		{name: "invalid heartbeat", opts: HandlerOptions{HeartbeatInterval: time.Microsecond, Inventory: smallInventory()}, want: "heartbeat"},
		{name: "unknown animation", opts: HandlerOptions{Animation: "missing", Inventory: smallInventory()}, want: `animation "missing" not found`},
		{name: "empty animation", opts: HandlerOptions{Inventory: animation.Inventory{"parrot": {}}}, want: "has no frames"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewHandlerWithOptions(tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	if _, err := NewHandlerWithOptions(HandlerOptions{Animation: "cat", Inventory: smallInventory()}); err != nil {
		t.Fatalf("valid custom animation: %v", err)
	}
	if NewHandler("test") == nil {
		t.Fatal("default handler is nil")
	}
}

func TestEventStreamOverHTTP(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{FrameDelay: time.Millisecond})
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	response := openEventStream(t, ctx, server.URL)
	defer closeResponseBody(t, response.Body)

	assertEventStreamHeaders(t, response)
	event := readFirstFrameEvent(t, response.Body)
	if event.Animation != "parrot" || event.Frame != "PARROT-A" || event.Index != 0 || event.Color != browserColors[0] {
		t.Errorf("unexpected first event: %+v", event)
	}
	cancel()
}

func TestEventStreamConcurrencyLimit(t *testing.T) {
	handler := newSmallHandler(t, HandlerOptions{FrameDelay: time.Millisecond, MaxStreams: 1})
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	first := openEventStream(t, ctx, server.URL)
	defer closeResponseBody(t, first.Body)

	second, err := http.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, second.Body)
	if second.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("second stream status = %d, want 503", second.StatusCode)
	}
	if second.Header.Get("Retry-After") != "1" {
		t.Errorf("Retry-After = %q, want 1", second.Header.Get("Retry-After"))
	}
	cancel()
}

func openEventStream(t *testing.T, ctx context.Context, baseURL string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func closeResponseBody(t *testing.T, body io.Closer) {
	t.Helper()
	if err := body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}

func assertEventStreamHeaders(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("Content-Type = %q", got)
	}
	if response.Header.Get("X-Accel-Buffering") != "no" {
		t.Errorf("X-Accel-Buffering = %q", response.Header.Get("X-Accel-Buffering"))
	}
}

func readFirstFrameEvent(t *testing.T, body io.Reader) frameEvent {
	t.Helper()
	scanner := bufio.NewScanner(body)
	var data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
		if line == "" && data != "" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	var event frameEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("decode event %q: %v", data, err)
	}
	return event
}

func TestStreamAnimationLifecycleAndErrors(t *testing.T) {
	anim := smallInventory()["parrot"]

	t.Run("streams and wraps frames", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var output bytes.Buffer
		flushes := 0
		err := streamAnimation(ctx, &output, nil, func() error {
			flushes++
			if flushes == 3 {
				cancel()
			}
			return nil
		}, "parrot", anim, time.Millisecond, time.Hour, false)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		text := output.String()
		for _, want := range []string{"PARROT-A", "PARROT-B", `"index":0`, `"index":1`, browserColors[0], browserColors[1]} {
			if !strings.Contains(text, want) {
				t.Errorf("stream does not contain %q: %s", want, text)
			}
		}
	})

	t.Run("mono color", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var output bytes.Buffer
		err := streamAnimation(ctx, &output, nil, func() error { cancel(); return nil }, "parrot", anim, time.Millisecond, time.Hour, true)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output.String(), `"color":"#d6e2f0"`) {
			t.Errorf("mono color missing: %s", output.String())
		}
	})

	t.Run("already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := streamAnimation(ctx, io.Discard, nil, nil, "parrot", anim, time.Millisecond, time.Hour, false)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})

	t.Run("empty frames", func(t *testing.T) {
		err := streamAnimation(context.Background(), io.Discard, nil, nil, "empty", animation.Animation{}, time.Millisecond, time.Hour, false)
		if err == nil || !strings.Contains(err.Error(), "no frames") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid delay", func(t *testing.T) {
		err := streamAnimation(context.Background(), io.Discard, nil, nil, "parrot", anim, 0, time.Hour, false)
		if err == nil || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid heartbeat", func(t *testing.T) {
		err := streamAnimation(context.Background(), io.Discard, nil, nil, "parrot", anim, time.Millisecond, 0, false)
		if err == nil || !strings.Contains(err.Error(), "heartbeat") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("prepare error", func(t *testing.T) {
		want := errors.New("prepare")
		err := streamAnimation(context.Background(), io.Discard, func() error { return want }, nil, "parrot", anim, time.Millisecond, time.Hour, false)
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	})

	t.Run("writer error", func(t *testing.T) {
		want := errors.New("write")
		err := streamAnimation(context.Background(), errorWriter{err: want}, nil, nil, "parrot", anim, time.Millisecond, time.Hour, false)
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	})

	t.Run("flush error", func(t *testing.T) {
		want := errors.New("flush")
		err := streamAnimation(context.Background(), io.Discard, nil, func() error { return want }, "parrot", anim, time.Millisecond, time.Hour, false)
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	})

	t.Run("heartbeat", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var output bytes.Buffer
		flushes := 0
		err := streamAnimation(ctx, &output, nil, func() error {
			flushes++
			if flushes == 2 {
				cancel()
			}
			return nil
		}, "parrot", anim, time.Hour, time.Millisecond, false)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if !strings.Contains(output.String(), ": keepalive\n\n") {
			t.Errorf("heartbeat is missing: %s", output.String())
		}
	})
}

func TestEventsRejectsNonFlushingWriter(t *testing.T) {
	writer := &basicResponseWriter{header: make(http.Header)}
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/events", nil)
	newSmallHandler(t, HandlerOptions{}).ServeHTTP(writer, req)
	if writer.status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", writer.status)
	}
	if !strings.Contains(writer.body.String(), "streaming unsupported") {
		t.Errorf("unexpected body: %s", writer.body.String())
	}
}

func TestEventsStreamsThroughUnwrapCapableWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	underlying := &cancelOnFlushWriter{
		ResponseRecorder: httptest.NewRecorder(),
		cancel:           cancel,
	}
	writer := unwrapResponseWriter{ResponseWriter: underlying}
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/events", nil).WithContext(ctx)

	newSmallHandler(t, HandlerOptions{}).ServeHTTP(writer, req)
	if underlying.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", underlying.Code, underlying.Body.String())
	}
	if got := underlying.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
}

func TestWriteTextResponseNewlineHandling(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/", nil)
	for _, frame := range []string{"", "A", "A\n"} {
		var output bytes.Buffer
		writeTextResponse(&output, req, "", frame, diagnosticOptions{})
		if !strings.Contains(output.String(), "\n\nHello from") {
			t.Errorf("frame %q did not end with one blank separator: %q", frame, output.String())
		}
		if strings.Contains(output.String(), "Version:") {
			t.Errorf("empty version was rendered: %q", output.String())
		}
		if !strings.HasSuffix(output.String(), "Project: "+projectURL+"\n") {
			t.Errorf("project is not the final line: %q", output.String())
		}
	}
}

func hasTextFramePrefix(body []byte, frames []string) bool {
	for _, frame := range frames {
		prefix := frame
		if prefix == "" || prefix[len(prefix)-1] != '\n' {
			prefix += "\n"
		}
		prefix += "\n"
		if bytes.HasPrefix(body, []byte(prefix)) {
			return true
		}
	}
	return false
}

func TestRunValidationListenFailureAndShutdown(t *testing.T) {
	if err := Run(context.Background(), Options{}); err == nil || !strings.Contains(err.Error(), "listen address is required") {
		t.Fatalf("empty address error = %v", err)
	}
	if err := Run(context.Background(), Options{Addr: "127.0.0.1:0", Animation: "missing", Stdout: io.Discard}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown animation error = %v", err)
	}
	if err := Run(context.Background(), Options{Addr: "invalid address", Stdout: io.Discard}); err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("invalid address error = %v", err)
	}
	var nilContext context.Context
	if err := Run(nilContext, Options{Addr: "invalid address", Stdout: io.Discard}); err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("nil context error = %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close occupied listener: %v", err)
		}
	}()
	if err := Run(context.Background(), Options{Addr: addr, Stdout: io.Discard}); err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("occupied address error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout bytes.Buffer
	if err := Run(ctx, Options{Addr: "127.0.0.1:0", Stdout: &stdout}); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}
	if !strings.Contains(stdout.String(), "hello HTTP server listening on") {
		t.Errorf("startup message missing: %s", stdout.String())
	}
}

func TestRunCancelsActiveEventStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ready := &readyWriter{line: make(chan string, 1)}
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{
			Addr:       "127.0.0.1:0",
			FrameDelay: time.Millisecond,
			Stdout:     ready,
		})
	}()

	var startup string
	select {
	case startup = <-ready.line:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("server did not report its listen address")
	}
	addr := strings.TrimSpace(strings.TrimPrefix(startup, "hello HTTP server listening on "))
	response, err := http.Get("http://" + addr + "/events")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer func() {
		// The server intentionally cancels this live response during shutdown.
		_ = response.Body.Close()
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server shutdown with an active stream: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not cancel the active stream during shutdown")
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type basicResponseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

type readyWriter struct {
	once sync.Once
	line chan string
}

type unwrapResponseWriter struct {
	http.ResponseWriter
}

func (w unwrapResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type cancelOnFlushWriter struct {
	*httptest.ResponseRecorder
	cancel context.CancelFunc
}

func (w *cancelOnFlushWriter) Flush() {
	w.ResponseRecorder.Flush()
	w.cancel()
}

func (w *readyWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { w.line <- string(p) })
	return len(p), nil
}

func (w *basicResponseWriter) Header() http.Header {
	return w.header
}

func (w *basicResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *basicResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}
