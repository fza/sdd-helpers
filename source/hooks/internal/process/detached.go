package process

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

// Detached starts a process in a session of its own, its output discarded,
// and lets it run on once the caller exits: a hook that started it returns at
// once, and Claude Code ending the hook's process group leaves it running.
type Detached struct{}

func (Detached) Start(_ context.Context, path string, args ...string) error {
	command := exec.Command(path, args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	err := command.Start()
	if err != nil {
		return fmt.Errorf("starting %s: %w", path, err)
	}

	err = command.Process.Release()
	if err != nil {
		return fmt.Errorf("releasing %s: %w", path, err)
	}

	return nil
}
