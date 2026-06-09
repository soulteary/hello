package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"
	"time"
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
	inv := Inventory{
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
	got := availableAnimations(Inventory{})
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
			anim := Animation{Frames: frames}

			var buf bytes.Buffer
			r := NewRenderer(&buf, true)

			runLoop(r, anim, loopOptions{
				loops:      tc.loops,
				frameDelay: time.Millisecond,
			})

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
	r := NewRenderer(&buf, true)
	runLoop(r, Animation{}, loopOptions{loops: 1, frameDelay: time.Millisecond})
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

// Test_RunLoop_StopChannelInterrupts asserts that closing opts.stop causes
// runLoop to return promptly even when loops=0 (infinite). It guards the
// SIGINT teardown path that main() relies on.
func Test_RunLoop_StopChannelInterrupts(t *testing.T) {
	frames := [][]byte{[]byte("x"), []byte("y")}
	anim := Animation{Frames: frames}

	stop := make(chan os.Signal, 1)
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)

	done := make(chan struct{})
	go func() {
		runLoop(r, anim, loopOptions{
			loops:      0,
			frameDelay: 10 * time.Millisecond,
			stop:       stop,
		})
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
