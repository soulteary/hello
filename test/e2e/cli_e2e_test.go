package e2e

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/hello/internal/animation"
)

// testBinary holds the path to a binary compiled once for the whole e2e suite.
// Building once (instead of `go run` per case) keeps the tests fast and, more
// importantly, lets us observe the program's real exit code rather than the
// exit code of the `go run` wrapper.
var testBinary string

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		// In -short mode the e2e tests skip themselves; don't pay the build cost.
		os.Exit(m.Run())
	}

	dir, err := os.MkdirTemp("", "hello-e2e")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	bin := filepath.Join(dir, "hello")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "github.com/soulteary/hello/cmd/hello")
	if out, err := build.CombinedOutput(); err != nil {
		panic("failed to build test binary: " + err.Error() + "\n" + string(out))
	}
	testBinary = bin

	os.Exit(m.Run())
}

// runCLI runs the pre-built test binary with the given args, returning its
// combined output and exit code.
func runCLI(t *testing.T, args ...string) (string, int) {
	t.Helper()
	if testBinary == "" {
		t.Skip("test binary not built")
	}
	cmd := exec.Command(testBinary, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v (output: %s)", err, out)
		}
	}
	return string(out), code
}

func Test_CLI_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	out, code := runCLI(t, "-version")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (output: %s)", code, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("expected version output, got empty string")
	}
}

func Test_CLI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	out, code := runCLI(t, "-list")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (output: %s)", code, out)
	}
	for _, name := range []string{"parrot", "pedro", "cat", "coffee", "loading"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected -list output to contain %q, got: %s", name, out)
		}
	}
}

func Test_CLI_UnknownAnimationExitsOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	out, code := runCLI(t, "-loops", "1", "definitely-not-an-animation")
	if code != 1 {
		t.Fatalf("expected exit code 1 for unknown animation, got %d (output: %s)", code, out)
	}
}

func Test_CLI_InvalidDelayExitsTwo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	out, code := runCLI(t, "-delay", "0")
	if code != 2 {
		t.Fatalf("expected exit code 2 for invalid delay, got %d (output: %s)", code, out)
	}
}

func Test_CLI_NegativeLoopsExitsTwo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	out, code := runCLI(t, "-loops", "-1")
	if code != 2 {
		t.Fatalf("expected exit code 2 for negative loops, got %d (output: %s)", code, out)
	}
}

func Test_CLI_RejectsExcessiveDelayAndExtraArguments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"-delay", "60001"}, want: "delay must be <= 60000 ms"},
		{args: []string{"cat", "pedro"}, want: "at most one animation"},
	} {
		out, code := runCLI(t, tc.args...)
		if code != 2 {
			t.Errorf("args %v: exit code = %d, want 2; output: %s", tc.args, code, out)
		}
		if !strings.Contains(out, tc.want) {
			t.Errorf("args %v: output does not contain %q: %s", tc.args, tc.want, out)
		}
	}
}

func Test_HTTP_ContentNegotiation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("signal-based HTTP shutdown is covered on Unix CI")
	}
	baseURL, stop := startHTTPServer(t)
	defer stop()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.5.0")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/plain") {
		t.Errorf("curl Content-Type = %q", response.Header.Get("Content-Type"))
	}
	if response.Header.Get("Project") != "https://github.com/soulteary/hello" {
		t.Errorf("curl Project header = %q", response.Header.Get("Project"))
	}
	if !bytes.Contains(body, []byte("Hello from soulteary/hello!\n\nVersion: ")) || !hasAnimationFramePrefix(body, animation.NewInventory()["parrot"].Frames) {
		t.Errorf("curl response is missing the parrot frame or diagnostics: %s", body)
	}
	if !bytes.HasSuffix(body, []byte("Project: https://github.com/soulteary/hello\n")) {
		t.Errorf("curl response does not end with the project URL: %s", body)
	}

	req, err = http.NewRequest(http.MethodHead, baseURL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.5.0")
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Project") != "https://github.com/soulteary/hello" {
		t.Errorf("curl HEAD Project header = %q", response.Header.Get("Project"))
	}

	req, err = http.NewRequest(http.MethodGet, baseURL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/html")
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(response.Body)
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") ||
		!bytes.Contains(body, []byte(`new EventSource(eventsURL)`)) ||
		!bytes.Contains(body, []byte(`Project: <a href="https://github.com/soulteary/hello">https://github.com/soulteary/hello</a>`)) {
		t.Errorf("browser response is not the animated HTML page: %s", body)
	}

	response, err = http.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	health, err := io.ReadAll(response.Body)
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(health) != "ok\n" {
		t.Errorf("health response = %d %q", response.StatusCode, health)
	}
}

func hasAnimationFramePrefix(body []byte, frames [][]byte) bool {
	for _, frame := range frames {
		prefix := make([]byte, 0, len(frame)+2)
		prefix = append(prefix, frame...)
		if len(prefix) == 0 || prefix[len(prefix)-1] != '\n' {
			prefix = append(prefix, '\n')
		}
		prefix = append(prefix, '\n')
		if bytes.HasPrefix(body, prefix) {
			return true
		}
	}
	return false
}

func startHTTPServer(t *testing.T) (string, func()) {
	t.Helper()
	cmd := exec.Command(testBinary, "-listen", "127.0.0.1:0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			lineCh <- scanner.Text()
			return
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		errCh <- io.EOF
	}()

	var line string
	select {
	case line = <-lineCh:
	case err := <-errCh:
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("read server address: %v; stderr: %s", err, stderr.String())
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("timed out waiting for HTTP server; stderr: %s", stderr.String())
	}
	const prefix = "hello HTTP server listening on "
	if !strings.HasPrefix(line, prefix) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("unexpected startup line %q; stderr: %s", line, stderr.String())
	}
	baseURL := "http://" + strings.TrimPrefix(line, prefix)

	stopped := false
	return baseURL, func() {
		if stopped {
			return
		}
		stopped = true
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			_ = cmd.Process.Kill()
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("HTTP server did not stop cleanly: %v; stderr: %s", err, stderr.String())
		}
	}
}
