// Command sdd-grounding-gate is the Claude Code hook holding a session to
// real reads of a project's sdd decision graph before it asks, drafts or captures.
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/internal/hookio"
	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
	"github.com/fza/sdd-helpers/source/hooks/internal/process"
	"github.com/fza/sdd-helpers/source/hooks/internal/sddcaps"
	"github.com/fza/sdd-helpers/source/hooks/internal/shellwalk"
)

func main() {
	working, err := os.Getwd()
	if err != nil {
		os.Exit(0)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		userHome = ""
	}

	executable, err := os.Executable()
	if err != nil {
		executable = ""
	}

	cacheRoot := os.Getenv("XDG_CACHE_HOME")
	if cacheRoot == "" && userHome != "" {
		cacheRoot = filepath.Join(userHome, ".cache")
	}

	env := grounding.Env{
		ProjectDir: os.Getenv("CLAUDE_PROJECT_DIR"),
		WorkingDir: working,
		Home:       userHome,
		Runner:     process.Runner{},
		Settings:   pluginconfig.Paths(userHome, runtime.GOOS, os.Getenv),
		Getenv:     os.Getenv,
		SddCache:   sddcaps.CacheDir(cacheRoot),
		LookPath:   exec.LookPath,
		Files:      shellwalk.OSFiles{},
		SddValues:  process.SddConfig{},
		Starter:    process.Detached{},
		Executable: executable,
		Processes:  process.Table{},
	}
	streams := hookio.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}

	os.Exit(grounding.Main(context.Background(), os.Args[1:], env, streams))
}
