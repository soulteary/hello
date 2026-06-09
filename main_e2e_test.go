package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	build := exec.Command("go", "build", "-o", bin, ".")
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
