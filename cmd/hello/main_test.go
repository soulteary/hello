package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/hello/internal/httpserver"
)

func TestRunVersionHelpAndList(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "version", args: []string{"-version"}, wantCode: 0, wantStdout: currentVersion()},
		{name: "help", args: []string{"-help"}, wantCode: 0, wantStderr: "Usage: hello"},
		{name: "list", args: []string{"-list"}, wantCode: 0, wantStdout: "parrot"},
		{name: "unknown flag", args: []string{"-unknown"}, wantCode: 2, wantStderr: "flag provided but not defined"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tc.args, &stdout, &stderr)
			if got != tc.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr: %s", got, tc.wantCode, stderr.String())
			}
			if tc.wantStdout != "" && !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Errorf("stdout does not contain %q: %s", tc.wantStdout, stdout.String())
			}
			if tc.wantStderr != "" && !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr does not contain %q: %s", tc.wantStderr, stderr.String())
			}
		})
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name          string
		stamped       string
		moduleVersion string
		want          string
	}{
		{name: "stamped wins", stamped: "v2.0.0", moduleVersion: "v1.0.0", want: "2.0.0"},
		{name: "module version for go install", stamped: "dev", moduleVersion: "v2.0.0", want: "2.0.0"},
		{name: "development build", stamped: "dev", moduleVersion: "(devel)", want: "dev"},
		{name: "empty fallback", want: "dev"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.stamped, tc.moduleVersion); got != tc.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q", tc.stamped, tc.moduleVersion, got, tc.want)
			}
		})
	}
}

func TestRunValidatesArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "too many positional args", args: []string{"cat", "pedro"}, want: "at most one animation"},
		{name: "zero delay", args: []string{"-delay", "0"}, want: "delay must be > 0"},
		{name: "negative delay", args: []string{"-delay", "-1"}, want: "delay must be > 0"},
		{name: "excessive delay", args: []string{"-delay", "60001"}, want: "delay must be <= 60000 ms"},
		{name: "negative loops", args: []string{"-loops", "-1"}, want: "loops must be >= 0"},
		{name: "bad integer", args: []string{"-delay", "nope"}, want: "invalid value"},
		{name: "listen with list", args: []string{"-listen", ":8080", "-list"}, want: "cannot be combined with list"},
		{name: "listen with loops", args: []string{"-listen", ":8080", "-loops", "1"}, want: "cannot be combined with loops"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			called := false
			code := runWithServer(tc.args, &stdout, &stderr, func(context.Context, httpserver.Options) error {
				called = true
				return nil
			})
			if code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr: %s", code, stderr.String())
			}
			if called {
				t.Fatal("server was called for invalid arguments")
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Errorf("stderr does not contain %q: %s", tc.want, stderr.String())
			}
		})
	}
}

func TestRunDispatchesTerminalMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-mono", "-loops", "1", "-delay", "1", "loading"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Loading") {
		t.Errorf("terminal output does not contain an animation frame: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\x1b[?25h") {
		t.Errorf("terminal output did not restore the cursor: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"does-not-exist"}, &stdout, &stderr); code != 1 {
		t.Fatalf("unknown animation exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("unknown animation error missing: %s", stderr.String())
	}
}

func TestRunDispatchesHTTPMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var got httpserver.Options
	code := runWithServer(
		[]string{"-listen", " 127.0.0.1:9090 ", "-a", "cat", "-delay", "120", "-mono"},
		&stdout,
		&stderr,
		func(ctx context.Context, opts httpserver.Options) error {
			if ctx == nil {
				t.Fatal("server context is nil")
			}
			got = opts
			return nil
		},
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if got.Addr != "127.0.0.1:9090" || got.Animation != "cat" || got.FrameDelay != 120*time.Millisecond || !got.Mono {
		t.Errorf("unexpected HTTP options: %+v", got)
	}
	if got.Stdout != &stdout {
		t.Error("HTTP stdout was not forwarded")
	}
}

func TestRunReportsHTTPFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	wantErr := errors.New("boom")
	code := runWithServer([]string{"-listen", ":8080"}, &stdout, &stderr, func(context.Context, httpserver.Options) error {
		return wantErr
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "HTTP server: boom") {
		t.Errorf("stderr does not contain server error: %s", stderr.String())
	}
}

func TestRunNilDependenciesAndDefaultServer(t *testing.T) {
	if code := runWithServer([]string{"-version"}, nil, nil, nil); code != 0 {
		t.Fatalf("version with nil writers returned %d", code)
	}

	var stderr bytes.Buffer
	code := runWithServer([]string{"-listen", "invalid address"}, nil, &stderr, nil)
	if code != 1 {
		t.Fatalf("invalid listen address exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "HTTP server:") {
		t.Errorf("missing HTTP server error: %s", stderr.String())
	}
}

func TestVersionWinsBeforeOtherValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-version", "-delay", "0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
}
