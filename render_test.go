package main

import (
	"bytes"
	"strings"
	"testing"
)

func twoFrameAnim() Animation {
	return Animation{
		Metadata: nil,
		Frames:   [][]byte{[]byte("AB"), []byte("CD")},
	}
}

func Test_Renderer_BeginEmitsHideAndClear(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.Begin()
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
	r.End()
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
	r.Draw(anim)
	if r.FrameIndex() != 1 {
		t.Errorf("expected frame index 1 after first Draw, got %d", r.FrameIndex())
	}
	r.Draw(anim)
	if r.FrameIndex() != 0 {
		t.Errorf("expected frame index to wrap to 0 after second Draw, got %d", r.FrameIndex())
	}
}

func Test_Renderer_DrawMonoOmitsSGRColor(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.Draw(twoFrameAnim())
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
	r.Draw(twoFrameAnim())
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
	r.Draw(Animation{})
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty animation, got %q", buf.String())
	}
	if r.FrameIndex() != 0 {
		t.Errorf("expected frame index to remain 0, got %d", r.FrameIndex())
	}
}
