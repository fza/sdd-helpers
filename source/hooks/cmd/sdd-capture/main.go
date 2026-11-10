// Command sdd-capture turns a draft file into an sdd graph entry: the
// project's lint, sdd's pre-flight, then `sdd new` with the body as one
// argument.
package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/fza/sdd-helpers/source/hooks/internal/capture"
	"github.com/fza/sdd-helpers/source/hooks/internal/hookio"
	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
	"github.com/fza/sdd-helpers/source/hooks/internal/process"
	"github.com/fza/sdd-helpers/source/hooks/internal/sddcaps"
)

func main() {
	working, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	cacheRoot := os.Getenv("XDG_CACHE_HOME")
	if cacheRoot == "" && home != "" {
		cacheRoot = filepath.Join(home, ".cache")
	}

	// Every command runs in a process group of its own, out of reach of the
	// terminal's interrupt, so the signal cancels the run and the runner kills
	// the group.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	env := capture.Env{
		WorkingDir: working,
		Runner:     process.Runner{},
		Limit:      os.Getenv(capture.LimitVariable),
		Pause:      capture.Sleep,
		Settings:   pluginconfig.Paths(home, runtime.GOOS, os.Getenv),
		Getenv:     os.Getenv,
		SddCache:   sddcaps.CacheDir(cacheRoot),
		LookPath:   exec.LookPath,
		Git:        process.Git{},
		TempRoot:   "/tmp",
		SddValues:  process.SddConfig{},
	}
	streams := hookio.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}

	code := capture.Main(ctx, os.Args[1:], env, streams)

	stop()
	os.Exit(code)
}
