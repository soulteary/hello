package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/soulteary/hello/internal/animation"
	"github.com/soulteary/hello/internal/render"
)

// UsageExamples is the example block appended to the flag usage output.
const UsageExamples = `
Examples:
  hello                      # play the default parrot animation
  hello pedro                # play the pedro animation (positional arg)
  hello -a cat               # play the cat animation (named flag)
  hello -animation coffee    # long form of -a
  hello -list                # list all available animations
  hello -mono -delay 120     # disable rainbow, slower frames
  hello -loops 3 pedro       # play pedro for 3 loops then exit
`

// defaultAnimation is played when no animation is requested.
const defaultAnimation = "parrot"

// Options bundles the resolved runtime configuration for Run. main is
// responsible for parsing flags and validating ranges; Run consumes the
// already-validated values and wires up the inventory, renderer and loop.
type Options struct {
	Animation string
	Loops     int
	Delay     time.Duration
	Mono      bool
	List      bool

	Stdout io.Writer
	Stderr io.Writer
}

// Run executes the CLI with the given options and returns a process exit
// code. It never calls os.Exit so that it stays testable; main translates
// the returned code into os.Exit.
func Run(opts Options) int {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	inventory := animation.NewInventory()

	if opts.List {
		for _, name := range availableAnimations(inventory) {
			if desc := inventory[name].Metadata["description"]; desc != "" {
				fmt.Fprintf(opts.Stdout, "%s\t%s\n", name, desc)
			} else {
				fmt.Fprintln(opts.Stdout, name)
			}
		}
		return 0
	}

	name := pickAnimationName(opts.Animation, nil, defaultAnimation)
	anim, ok := inventory[name]
	if !ok {
		fmt.Fprintf(opts.Stderr, "Animation %q not found. Available: %v\n",
			name, availableAnimations(inventory))
		return 1
	}

	renderer := render.NewRenderer(opts.Stdout, opts.Mono)
	stop, cleanup := installSignalHandler()
	defer cleanup()

	renderer.Begin()
	defer renderer.End()

	runLoop(renderer, anim, loopOptions{
		loops:      opts.Loops,
		frameDelay: opts.Delay,
		stop:       stop,
	})
	return 0
}

// availableAnimations returns the sorted list of animation names present in
// the inventory. Sorting keeps the -list output and error messages stable
// across runs and platforms.
func availableAnimations(inv animation.Inventory) []string {
	names := make([]string, 0, len(inv))
	for name := range inv {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

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

// ResolveAnimation collapses the flag and positional argument sources into a
// single animation name, applying the precedence: explicit -a/-animation flag
// > first positional arg. An empty result defers to Run's default. main calls
// this so Run can take a single already-resolved name.
func ResolveAnimation(flagValue string, args []string) string {
	if flagValue != "" {
		return flagValue
	}
	if len(args) > 0 {
		return args[0]
	}
	return ""
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
// this function signals the caller to tear the renderer down.
func runLoop(renderer *render.Renderer, anim animation.Animation, opts loopOptions) {
	frames := len(anim.Frames)
	if frames == 0 {
		return
	}

	ticker := time.NewTicker(opts.frameDelay)
	defer ticker.Stop()

	renderer.Draw(anim)
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
			renderer.Draw(anim)
			renderer.AdvanceColor()
			drawn++
		}
	}
}
