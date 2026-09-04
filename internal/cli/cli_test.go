package cli

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/hello/internal/animation"
	"github.com/soulteary/hello/internal/render"
)

func Test_SelectAnimation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		def  string
		want string
	}{
		{"empty args returns default", nil, "parrot", "parrot"},
		{"empty string arg returns default", []string{""}, "parrot", "parrot"},
		{"first arg wins", []string{"pedro", "ignored"}, "parrot", "pedro"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectAnimation(tc.args, tc.def); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func Test_ResolveAnimation(t *testing.T) {
	tests := []struct {
		flagValue string
		args      []string
		want      string
	}{
		{flagValue: "cat", args: []string{"pedro"}, want: "cat"},
		{args: []string{"pedro"}, want: "pedro"},
		{args: []string{""}, want: ""},
		{want: ""},
	}
	for _, tc := range tests {
		if got := ResolveAnimation(tc.flagValue, tc.args); got != tc.want {
			t.Errorf("ResolveAnimation(%q, %v) = %q, want %q", tc.flagValue, tc.args, got, tc.want)
		}
	}
}

func Test_RunValidation(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
		code int
	}{
		{name: "zero delay", opts: Options{Delay: 0}, want: "delay must be > 0", code: 2},
		{name: "negative delay", opts: Options{Delay: -time.Millisecond}, want: "delay must be > 0", code: 2},
		{name: "excessive delay", opts: Options{Delay: MaxFrameDelay + time.Millisecond}, want: "delay must be <=", code: 2},
		{name: "negative loops", opts: Options{Delay: time.Millisecond, Loops: -1}, want: "loops must be >= 0", code: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			tc.opts.Stdout = &stdout
			tc.opts.Stderr = &stderr
			if got := Run(tc.opts); got != tc.code {
				t.Fatalf("exit code = %d, want %d", got, tc.code)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Errorf("stderr does not contain %q: %s", tc.want, stderr.String())
			}
		})
	}
}

func Test_RunListUnknownAndSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(Options{List: true, Delay: time.Millisecond, Stdout: &stdout, Stderr: &stderr}); code != 0 {
		t.Fatalf("list exit code = %d, want 0", code)
	}
	for _, want := range []string{"cat\tA bouncing cat", "parrot\tThe classic Party Parrot."} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("list output does not contain %q: %s", want, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(Options{Animation: "missing", Delay: time.Millisecond, Stdout: &stdout, Stderr: &stderr}); code != 1 {
		t.Fatalf("unknown animation exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Available: [cat coffee loading parrot pedro]") {
		t.Errorf("unknown-animation output is not stable: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(Options{Loops: 1, Delay: time.Millisecond, Mono: true, Stdout: &stdout, Stderr: &stderr}); code != 0 {
		t.Fatalf("default animation exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "\x1b[?25l") || !strings.Contains(stdout.String(), "\x1b[?25h") {
		t.Errorf("successful run did not bracket output with cursor controls: %q", stdout.String())
	}
}

func Test_RunStopsOnOutputFailure(t *testing.T) {
	want := errors.New("closed pipe")
	stderr := &bytes.Buffer{}
	if code := Run(Options{
		Loops:  1,
		Delay:  time.Millisecond,
		Mono:   true,
		Stdout: failingWriter{err: want},
		Stderr: stderr,
	}); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "write output: closed pipe") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	if code := Run(Options{
		List:   true,
		Delay:  time.Millisecond,
		Stdout: failingWriter{err: want},
		Stderr: stderr,
	}); code != 1 {
		t.Fatalf("list exit code = %d, want 1", code)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func Test_PickAnimationName(t *testing.T) {
	cases := []struct {
		name      string
		flagValue string
		args      []string
		def       string
		want      string
	}{
		{"flag wins over positional", "cat", []string{"pedro"}, "parrot", "cat"},
		{"flag wins over default", "cat", nil, "parrot", "cat"},
		{"empty flag falls back to positional", "", []string{"pedro"}, "parrot", "pedro"},
		{"empty flag and empty args fall back to default", "", nil, "parrot", "parrot"},
		{"empty flag and empty positional fall back to default", "", []string{""}, "parrot", "parrot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickAnimationName(tc.flagValue, tc.args, tc.def); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func Test_AvailableAnimations_SortedAndComplete(t *testing.T) {
	inv := animation.Inventory{
		"pedro":  {},
		"cat":    {},
		"parrot": {},
	}
	got := availableAnimations(inv)
	want := []string{"cat", "parrot", "pedro"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_AvailableAnimations_Empty(t *testing.T) {
	got := availableAnimations(animation.Inventory{})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// Test_RunLoop_DrawsExactlyLoopsTimesFrames asserts that runLoop with
// loops=N draws exactly N*len(Frames) frames before returning, regardless
// of the frame count divisibility.
func Test_RunLoop_DrawsExactlyLoopsTimesFrames(t *testing.T) {
	cases := []struct {
		name   string
		frames int
		loops  int
	}{
		{"2 frames, 3 loops", 2, 3},
		{"5 frames, 1 loop", 5, 1},
		{"3 frames, 4 loops", 3, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frames := make([][]byte, tc.frames)
			for i := range frames {
				frames[i] = []byte{'x'}
			}
			anim := animation.Animation{Frames: frames}

			var buf bytes.Buffer
			r := render.NewRenderer(&buf, true)

			if err := runLoop(r, anim, loopOptions{
				loops:      tc.loops,
				frameDelay: time.Millisecond,
			}); err != nil {
				t.Fatal(err)
			}

			want := tc.loops * tc.frames
			// FrameIndex wraps; we infer total draws from byte count instead:
			// each frame writes ansiHome + 'x' + ansiClearEOL, so count 'x'.
			got := bytes.Count(buf.Bytes(), []byte{'x'})
			if got != want {
				t.Errorf("expected %d frames drawn, got %d", want, got)
			}
		})
	}
}

func Test_RunLoop_EmptyAnimationReturnsImmediately(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewRenderer(&buf, true)
	if err := runLoop(r, animation.Animation{}, loopOptions{loops: 1, frameDelay: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

// Test_RunLoop_StopChannelInterrupts asserts that closing opts.stop causes
// runLoop to return promptly even when loops=0 (infinite). It guards the
// SIGINT teardown path that main() relies on.
func Test_RunLoop_StopChannelInterrupts(t *testing.T) {
	frames := [][]byte{[]byte("x"), []byte("y")}
	anim := animation.Animation{Frames: frames}

	stop := make(chan os.Signal, 1)
	var buf bytes.Buffer
	r := render.NewRenderer(&buf, true)

	done := make(chan struct{})
	go func() {
		if err := runLoop(r, anim, loopOptions{
			loops:      0,
			frameDelay: 10 * time.Millisecond,
			stop:       stop,
		}); err != nil {
			t.Errorf("runLoop: %v", err)
		}
		close(done)
	}()

	// Give runLoop a chance to draw at least the first frame, then signal.
	time.Sleep(20 * time.Millisecond)
	close(stop)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runLoop did not exit after stop channel was closed")
	}
}

func Test_InstallSignalHandler_Cleanup(t *testing.T) {
	ch, cleanup := installSignalHandler()
	if ch == nil {
		t.Fatal("expected non-nil signal channel")
	}
	cleanup() // must not panic and must be idempotent w.r.t. signal.Stop
	cleanup()
}
