package render

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/soulteary/hello/internal/animation"
)

func twoFrameAnim() animation.Animation {
	return animation.Animation{
		Metadata: nil,
		Frames:   [][]byte{[]byte("AB"), []byte("CD")},
	}
}

func Test_Renderer_BeginEmitsHideAndClear(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	if err := r.Begin(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[?25l") {
		t.Errorf("expected hide-cursor sequence in Begin output, got %q", out)
	}
	if !strings.Contains(out, "\x1b[2J") {
		t.Errorf("expected clear-screen sequence in Begin output, got %q", out)
	}
}

func Test_Renderer_EndShowsCursor(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	if err := r.End(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[?25h") {
		t.Errorf("expected show-cursor sequence in End output, got %q", out)
	}
}

func Test_Renderer_DrawAdvancesAndWraps(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	anim := twoFrameAnim()

	if r.FrameIndex() != 0 {
		t.Fatalf("expected initial frame index 0, got %d", r.FrameIndex())
	}
	if err := r.Draw(anim); err != nil {
		t.Fatal(err)
	}
	if r.FrameIndex() != 1 {
		t.Errorf("expected frame index 1 after first Draw, got %d", r.FrameIndex())
	}
	if err := r.Draw(anim); err != nil {
		t.Fatal(err)
	}
	if r.FrameIndex() != 0 {
		t.Errorf("expected frame index to wrap to 0 after second Draw, got %d", r.FrameIndex())
	}
}

func Test_Renderer_DrawNormalizesNegativeFrameIndex(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.frameIdx = -1
	if err := r.Draw(twoFrameAnim()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "CD") {
		t.Errorf("negative frame index did not wrap to the last frame: %q", buf.String())
	}
	if r.FrameIndex() != 0 {
		t.Errorf("frame index = %d, want 0", r.FrameIndex())
	}
}

func Test_Renderer_DrawClearsEveryLine(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	if err := r.Draw(animation.Animation{Frames: [][]byte{[]byte("A\nB")}}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), ansiClearEOL); got != 2 {
		t.Errorf("clear-to-EOL count = %d, want 2; output: %q", got, buf.String())
	}
}

func Test_Renderer_DrawMonoOmitsSGRColor(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	if err := r.Draw(twoFrameAnim()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "\x1b[38;5;") {
		t.Errorf("mono mode should not emit 256-color SGR, got %q", out)
	}
	if !strings.Contains(out, "AB") {
		t.Errorf("expected first frame content 'AB' in output, got %q", out)
	}
}

func Test_Renderer_DrawColorEmitsFirstPaletteEntry(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	if err := r.Draw(twoFrameAnim()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[38;5;") {
		t.Errorf("non-mono mode should emit 256-color SGR, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;210m") {
		t.Errorf("expected first palette color 210 in output, got %q", out)
	}
}

func Test_Renderer_AdvanceColorWrapsAround(t *testing.T) {
	r := NewRenderer(&bytes.Buffer{}, false)
	for i := 0; i < len(colors); i++ {
		r.AdvanceColor()
	}
	if r.ColorIndex() != 0 {
		t.Errorf("expected color index to wrap to 0 after %d advances, got %d", len(colors), r.ColorIndex())
	}
}

func Test_Renderer_AdvanceColorNoOpInMono(t *testing.T) {
	r := NewRenderer(&bytes.Buffer{}, true)
	r.AdvanceColor()
	r.AdvanceColor()
	if r.ColorIndex() != 0 {
		t.Errorf("expected color index 0 in mono mode, got %d", r.ColorIndex())
	}
}

func Test_Renderer_DrawEmptyAnimationIsNoop(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	if err := r.Draw(animation.Animation{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty animation, got %q", buf.String())
	}
	if r.FrameIndex() != 0 {
		t.Errorf("expected frame index to remain 0, got %d", r.FrameIndex())
	}
}

func Test_Renderer_PropagatesWriteErrors(t *testing.T) {
	want := errors.New("closed output")
	r := NewRenderer(failingWriter{err: want}, true)

	for name, call := range map[string]func() error{
		"begin": func() error { return r.Begin() },
		"draw":  func() error { return r.Draw(twoFrameAnim()) },
		"end":   func() error { return r.End() },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
		})
	}

	short := NewRenderer(shortWriter{}, true)
	if err := short.Draw(twoFrameAnim()); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write error = %v, want io.ErrShortWrite", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func BenchmarkRenderer_Draw(b *testing.B) {
	anim := twoFrameAnim()
	for _, mono := range []bool{true, false} {
		name := "color"
		if mono {
			name = "mono"
		}
		b.Run(name, func(b *testing.B) {
			r := NewRenderer(io.Discard, mono)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := r.Draw(anim); err != nil {
					b.Fatal(err)
				}
				r.AdvanceColor()
			}
		})
	}
}
