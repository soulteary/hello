package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// selectAnimation picks the animation name to play. The first non-empty
// positional argument wins; otherwise the default name is returned. The
// empty-string fallback is preserved so that an explicit `""` still defers
// to the default (handy for callers that pass through user input verbatim).
func selectAnimation(args []string, defaultName string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return defaultName
}

// pickAnimationName resolves the final animation name from all sources, with
// precedence: explicit -a/-animation flag > first positional arg > default.
// An empty flag value is treated as "unset" so users can still rely on the
// positional argument when the flag is absent.
func pickAnimationName(flagValue string, args []string, defaultName string) string {
	if flagValue != "" {
		return flagValue
	}
	return selectAnimation(args, defaultName)
}

// loopOptions bundles the runtime configuration for runLoop.
//
// stop is an optional channel that, when closed or sent to, causes runLoop
// to return at the next tick boundary. Tests inject a controllable channel;
// production code typically leaves it nil and lets installSignalHandler
// wire SIGINT/SIGTERM through registerSignalHandler instead.
type loopOptions struct {
	loops      int
	frameDelay time.Duration
	stop       <-chan os.Signal
}

// installSignalHandler registers SIGINT/SIGTERM forwarding and returns the
// channel plus a cleanup func. It is intentionally split from runLoop so the
// signal subscription happens *before* the renderer hides the cursor — that
// way Ctrl+C during the tiny window before runLoop starts still triggers the
// deferred renderer.End() in main and the terminal cursor is restored.
func installSignalHandler() (<-chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	return ch, func() { signal.Stop(ch) }
}

// runLoop drives the animation: it ticks the renderer at frameDelay,
// advances the rainbow color on each tick, and exits cleanly when opts.stop
// fires (or when the configured number of loops is reached). Returning from
// this function signals main() to tear the renderer down.
func runLoop(renderer *Renderer, animation Animation, opts loopOptions) {
	frames := len(animation.Frames)
	if frames == 0 {
		return
	}

	ticker := time.NewTicker(opts.frameDelay)
	defer ticker.Stop()

	renderer.Draw(animation)
	renderer.AdvanceColor()
	drawn := 1

	// Total frames to draw before exiting; 0 means run forever.
	target := 0
	if opts.loops > 0 {
		target = opts.loops * frames
	}

	for {
		if target > 0 && drawn >= target {
			return
		}
		select {
		case <-opts.stop:
			return
		case <-ticker.C:
			renderer.Draw(animation)
			renderer.AdvanceColor()
			drawn++
		}
	}
}
