package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootReflectsProxyIdentityWithoutCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://hello.example/", nil)
	req.Header.Set("X-Forwarded-User", "test-admin-001")
	req.Header.Set("X-Auth-Email", "admin@example.com")
	req.Header.Set("Authorization", "Bearer must-not-leak")
	req.Header.Set("Cookie", "session=must-not-leak")
	req.Header.Set("X-Auth-Request-Access-Token", "must-not-leak-auth-token")
	req.Header.Set("X-Forwarded-Access-Token", "must-not-leak-forwarded-token")
	recorder := httptest.NewRecorder()

	NewHandler("1.1.0").ServeHTTP(recorder, req)
	response := recorder.Result()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	text := string(body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.StatusCode, http.StatusOK, text)
	}
	for _, want := range []string{
		"Hello from soulteary/hello!",
		"Version: 1.1.0",
		"X-Forwarded-User: test-admin-001",
		"X-Auth-Email: admin@example.com",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("response does not contain %q: %s", want, text)
		}
	}
	for _, secret := range []string{"must-not-leak", "Authorization:", "Cookie:"} {
		if strings.Contains(text, secret) {
			t.Errorf("response leaked %q: %s", secret, text)
		}
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestHealthAndMethodContracts(t *testing.T) {
	handler := NewHandler("test")
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "health get", method: http.MethodGet, path: "/healthz", status: http.StatusOK},
		{name: "health head", method: http.MethodHead, path: "/healthz", status: http.StatusOK},
		{name: "root head", method: http.MethodHead, path: "/", status: http.StatusOK},
		{name: "root post", method: http.MethodPost, path: "/", status: http.StatusMethodNotAllowed},
		{name: "missing", method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, "http://hello.example"+tc.path, nil))
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tc.status, recorder.Body.String())
			}
		})
	}
}

func TestSafeReflectedHeader(t *testing.T) {
	tests := map[string]bool{
		"X-Forwarded-User":                true,
		"X-Forwarded-For":                 true,
		"X-Forwarded-Proto":               true,
		"X-Auth-User":                     true,
		"X-Auth-Email":                    true,
		"X-Real-IP":                       true,
		"User-Agent":                      true,
		"Authorization":                   false,
		"Cookie":                          false,
		"Proxy-Authorization":             false,
		"X-Auth-Request-Access-Token":     false,
		"X-Forwarded-Access-Token":        false,
		"X-Forwarded-Authorization":       false,
		"X-Auth-Unrecognized-Information": false,
	}
	for name, want := range tests {
		if got := safeReflectedHeader(name); got != want {
			t.Errorf("safeReflectedHeader(%q) = %t, want %t", name, got, want)
		}
	}
}
