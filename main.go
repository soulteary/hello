package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"
)

// version is overridden at build time via -ldflags "-X main.version=...".
// "dev" is a sensible default for `go run` and unstamped local builds.
var version = "dev"

const usageExamples = `
Examples:
  hello                      # play the default parrot animation
  hello pedro                # play the pedro animation (positional arg)
  hello -a cat               # play the cat animation (named flag)
  hello -animation coffee    # long form of -a
  hello -list                # list all available animations
  hello -mono -delay 120     # disable rainbow, slower frames
  hello -loops 3 pedro       # play pedro for 3 loops then exit
`

func main() {
	loops := flag.Int("loops", 0, "number of times to loop (default: infinite)")
	mono := flag.Bool("mono", false, "disable rainbow colors")
	delay := flag.Int("delay", 75, "frame delay in ms (must be > 0)")
	list := flag.Bool("list", false, "list available animations and exit")
	showVersion := flag.Bool("version", false, "print version and exit")

	var animationFlag string
	flag.StringVar(&animationFlag, "animation", "", "animation name to play (overrides positional argument)")
	flag.StringVar(&animationFlag, "a", "", "animation name to play (shorthand for -animation)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [animation]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), usageExamples)
	}

	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if *delay <= 0 {
		fmt.Fprintln(os.Stderr, "delay must be > 0")
		os.Exit(2)
	}

	// Guard against runaway frame delays (a typo like -delay 999999 would
	// otherwise leave the user staring at a frozen frame for ~17 minutes).
	const maxDelayMs = 60_000
	if *delay > maxDelayMs {
		fmt.Fprintf(os.Stderr, "delay must be <= %d ms\n", maxDelayMs)
		os.Exit(2)
	}

	if *loops < 0 {
		fmt.Fprintln(os.Stderr, "loops must be >= 0")
		os.Exit(2)
	}

	inventory := NewInventory()

	if *list {
		for _, name := range availableAnimations(inventory) {
			if desc := inventory[name].Metadata["description"]; desc != "" {
				fmt.Printf("%s\t%s\n", name, desc)
			} else {
				fmt.Println(name)
			}
		}
		return
	}

	animationName := pickAnimationName(animationFlag, flag.Args(), "parrot")

	animation, ok := inventory[animationName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Animation %q not found. Available: %v\n",
			animationName, availableAnimations(inventory))
		os.Exit(1)
	}

	renderer := NewRenderer(os.Stdout, *mono)
	stop, cleanup := installSignalHandler()
	defer cleanup()

	renderer.Begin()
	defer renderer.End()

	runLoop(renderer, animation, loopOptions{
		loops:      *loops,
		frameDelay: time.Duration(*delay) * time.Millisecond,
		stop:       stop,
	})
}

// availableAnimations returns the sorted list of animation names present in
// the inventory. Sorting keeps the -list output and error messages stable
// across runs and platforms.
func availableAnimations(inv Inventory) []string {
	names := make([]string, 0, len(inv))
	for name := range inv {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
