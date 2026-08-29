package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/soulteary/hello/internal/cli"
	"github.com/soulteary/hello/internal/httpserver"
)

// version is overridden at build time via -ldflags "-X main.version=...".
// "dev" is a sensible default for `go run` and unstamped local builds.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses flags, validates them and dispatches to cli.Run. It returns a
// process exit code so it stays testable and main can stay a one-liner.
func run(args []string) int {
	fs := flag.NewFlagSet("hello", flag.ExitOnError)

	loops := fs.Int("loops", 0, "number of times to loop (default: infinite)")
	mono := fs.Bool("mono", false, "disable rainbow colors")
	delay := fs.Int("delay", 75, "frame delay in ms (must be > 0)")
	list := fs.Bool("list", false, "list available animations and exit")
	listen := fs.String("listen", "", "serve HTTP on this address instead of playing an animation (for example, :8080)")
	showVersion := fs.Bool("version", false, "print version and exit")

	var animationFlag string
	fs.StringVar(&animationFlag, "animation", "", "animation name to play (overrides positional argument)")
	fs.StringVar(&animationFlag, "a", "", "animation name to play (shorthand for -animation)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags] [animation]\n\nFlags:\n", os.Args[0])
		fs.PrintDefaults()
		fmt.Fprint(fs.Output(), cli.UsageExamples)
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Println(version)
		return 0
	}

	if addr := strings.TrimSpace(*listen); addr != "" {
		if *list {
			fmt.Fprintln(os.Stderr, "listen cannot be combined with list")
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := httpserver.Run(ctx, httpserver.Options{Addr: addr, Version: version, Stdout: os.Stdout}); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server: %v\n", err)
			return 1
		}
		return 0
	}

	if *delay <= 0 {
		fmt.Fprintln(os.Stderr, "delay must be > 0")
		return 2
	}

	// Guard against runaway frame delays (a typo like -delay 999999 would
	// otherwise leave the user staring at a frozen frame for ~17 minutes).
	const maxDelayMs = 60_000
	if *delay > maxDelayMs {
		fmt.Fprintf(os.Stderr, "delay must be <= %d ms\n", maxDelayMs)
		return 2
	}

	if *loops < 0 {
		fmt.Fprintln(os.Stderr, "loops must be >= 0")
		return 2
	}

	return cli.Run(cli.Options{
		Animation: cli.ResolveAnimation(animationFlag, fs.Args()),
		Loops:     *loops,
		Delay:     time.Duration(*delay) * time.Millisecond,
		Mono:      *mono,
		List:      *list,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
}
