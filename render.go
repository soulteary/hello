package main

import (
	"bytes"
	"fmt"
	"io"
)

// ANSI control sequences used by the renderer. They are kept as exported
// constants so tests can assert on the raw bytes without re-deriving them.
const (
	ansiHideCursor = "\x1b[?25l"
	ansiShowCursor = "\x1b[?25h"
	ansiClear      = "\x1b[2J"
	ansiHome       = "\x1b[H"
	ansiReset      = "\x1b[0m"
	ansiClearEOL   = "\x1b[K"
)

// Renderer streams animation frames to an io.Writer using ANSI escape
// sequences. It owns the frame and color cursors so that color cycling can
// advance independently of frame advancement (callers decide when to call
// AdvanceColor).
//
// A Renderer is not safe for concurrent use: all methods mutate the internal
// cursors without synchronization, so a single goroutine must drive it.
type Renderer struct {
	out      io.Writer
	mono     bool
	colorIdx int
	frameIdx int
}

// NewRenderer constructs a Renderer writing to out. When mono is true, frames
// are emitted without SGR color sequences and AdvanceColor becomes a no-op.
func NewRenderer(out io.Writer, mono bool) *Renderer {
	return &Renderer{out: out, mono: mono}
}

// Begin hides the cursor, clears the screen and homes the cursor. It should
// be called once before the first Draw.
func (r *Renderer) Begin() {
	fmt.Fprint(r.out, ansiHideCursor+ansiClear+ansiHome)
}

// End restores the cursor and resets attributes. Safe to call from a defer
// even if Begin was never executed.
func (r *Renderer) End() {
	fmt.Fprint(r.out, ansiShowCursor+ansiReset+"\n")
}

// Draw renders the next frame of animation to the writer. The frame is
// selected by the renderer's internal frame cursor (modulo the frame count)
// and the cursor is advanced after a successful write. In color mode the
// active palette entry wraps around the cursor on every line; the caller is
// responsible for invoking AdvanceColor on the desired cadence.
func (r *Renderer) Draw(animation Animation) {
	if len(animation.Frames) == 0 {
		return
	}
	idx := r.frameIdx % len(animation.Frames)
	if idx < 0 {
		idx += len(animation.Frames)
	}
	frame := animation.Frames[idx]
	lines := bytes.Split(frame, []byte{'\n'})

	var buf bytes.Buffer
	buf.WriteString(ansiHome)

	colored := !r.mono
	color := 0
	if colored {
		color = colors[r.colorIdx%len(colors)]
	}

	for i, line := range lines {
		if colored {
			fmt.Fprintf(&buf, "\x1b[38;5;%dm", color)
		}
		buf.Write(line)
		buf.WriteString(ansiClearEOL)
		if colored {
			buf.WriteString(ansiReset)
		}
		if i < len(lines)-1 {
			buf.WriteByte('\n')
		}
	}

	_, _ = r.out.Write(buf.Bytes())
	r.frameIdx++
	if r.frameIdx >= len(animation.Frames) {
		r.frameIdx = 0
	}
}

// AdvanceColor moves to the next palette entry. No-op when mono is true.
func (r *Renderer) AdvanceColor() {
	if r.mono {
		return
	}
	r.colorIdx++
	if r.colorIdx >= len(colors) {
		r.colorIdx = 0
	}
}

// FrameIndex returns the frame cursor that will be used by the next Draw.
func (r *Renderer) FrameIndex() int { return r.frameIdx }

// ColorIndex returns the current palette cursor.
func (r *Renderer) ColorIndex() int { return r.colorIdx }
