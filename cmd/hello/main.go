package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/soulteary/hello/internal/cli"
	"github.com/soulteary/hello/internal/httpserver"
)

// version is overridden at build time via -ldflags "-X main.version=...".
// "dev" is a sensible default for `go run` and unstamped local builds.
var version = "dev"
var exitProcess = os.Exit

func currentVersion() string {
	info, _ := debug.ReadBuildInfo()
	return resolveVersion(version, buildInfoVersion(info))
}

func buildInfoVersion(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}
	return info.Main.Version
}

func resolveVersion(stamped, moduleVersion string) string {
	if stamped != "" && stamped != "dev" {
		return strings.TrimPrefix(stamped, "v")
	}
	if moduleVersion != "" && moduleVersion != "(devel)" {
		return strings.TrimPrefix(moduleVersion, "v")
	}
	if stamped == "" {
		return "dev"
	}
	return stamped
}

func main() {
	exitProcess(run(os.Args[1:], os.Stdout, os.Stderr))
}

type serverRunner func(context.Context, httpserver.Options) error

// run parses flags, validates them and dispatches to the terminal or HTTP
// mode. It returns a process exit code so main can stay a one-liner.
func run(args []string, stdout, stderr io.Writer) int {
	return runWithServer(args, stdout, stderr, httpserver.Run)
}

func runWithServer(args []string, stdout, stderr io.Writer, serve serverRunner) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if serve == nil {
		serve = httpserver.Run
	}

	fs := flag.NewFlagSet("hello", flag.ContinueOnError)
	fs.SetOutput(stderr)

	loops := fs.Int("loops", 0, "number of terminal loops (default: infinite; unavailable with -listen)")
	mono := fs.Bool("mono", false, "disable rainbow colors")
	delay := fs.Int("delay", 75, "terminal/SSE frame delay in ms (1-60000)")
	list := fs.Bool("list", false, "list available animations and exit")
	listen := fs.String("listen", "", "serve HTTP on this address instead of playing an animation (for example, :8080)")
	maxStreams := fs.Int("http-max-streams", 64, "maximum concurrent SSE streams (unavailable without -listen)")
	reflectQuery := fs.Bool("reflect-query", false, "include the raw URL query in text diagnostics (unavailable without -listen)")
	reflectIdentity := fs.Bool("reflect-identity", false, "include allowlisted identity headers in text diagnostics (unavailable without -listen)")
	reflectHostname := fs.Bool("reflect-hostname", true, "include the container hostname in text diagnostics (unavailable without -listen)")
	showVersion := fs.Bool("version", false, "print version and exit")

	var animationFlag string
	fs.StringVar(&animationFlag, "animation", "", "animation name to play or serve (overrides positional argument)")
	fs.StringVar(&animationFlag, "a", "", "animation name to play (shorthand for -animation)")

	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage: hello [flags] [animation]\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprint(fs.Output(), cli.UsageExamples)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, currentVersion())
		return 0
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(stderr, "expected at most one animation name, got %d\n", fs.NArg())
		return 2
	}
	if *delay <= 0 {
		fmt.Fprintln(stderr, "delay must be > 0")
		return 2
	}
	if *delay > int(cli.MaxFrameDelay/time.Millisecond) {
		fmt.Fprintf(stderr, "delay must be <= %d ms\n", cli.MaxFrameDelay/time.Millisecond)
		return 2
	}
	if *loops < 0 {
		fmt.Fprintln(stderr, "loops must be >= 0")
		return 2
	}

	animationName := cli.ResolveAnimation(animationFlag, fs.Args())

	if addr := strings.TrimSpace(*listen); addr != "" {
		if *list {
			fmt.Fprintln(stderr, "listen cannot be combined with list")
			return 2
		}
		if *loops != 0 {
			fmt.Fprintln(stderr, "listen cannot be combined with loops")
			return 2
		}
		if *maxStreams <= 0 {
			fmt.Fprintln(stderr, "http-max-streams must be > 0")
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := serve(ctx, httpserver.Options{
			Addr:            addr,
			Version:         currentVersion(),
			Animation:       animationName,
			FrameDelay:      time.Duration(*delay) * time.Millisecond,
			Mono:            *mono,
			MaxStreams:      *maxStreams,
			ReflectQuery:    *reflectQuery,
			ReflectIdentity: *reflectIdentity,
			ReflectHostname: *reflectHostname,
			Stdout:          stdout,
		}); err != nil {
			fmt.Fprintf(stderr, "HTTP server: %v\n", err)
			return 1
		}
		return 0
	}

	return cli.Run(cli.Options{
		Animation: animationName,
		Loops:     *loops,
		Delay:     time.Duration(*delay) * time.Millisecond,
		Mono:      *mono,
		List:      *list,
		Stdout:    stdout,
		Stderr:    stderr,
	})
}
