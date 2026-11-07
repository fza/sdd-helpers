// Package releaseprune deletes the release folders of the plugin's data
// directory no session may still run. The launcher, plugin/hooks/run, unpacks
// each release into a folder of its own and never removes one, so without the
// prune every release would stay for good. A run starts at most once a day,
// detached, so no hook waits for it; each run records its outcome, and a
// failed one is told once.
package releaseprune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fza/sdd-helpers/source/hooks/internal/atomicfile"
	"github.com/fza/sdd-helpers/source/hooks/internal/filelock"
	"github.com/fza/sdd-helpers/source/hooks/internal/safefs"
)

// StatusName is the run's record in the data directory: when the last run
// started and finished, what it deleted, and its error.
const StatusName = "release-prune.json"

const (
	lockName    = "release-prune.lock"
	runLockName = "release-prune.run.lock"
	// retryDenied is how soon a run that was denied something is claimed
	// again, so a sandboxed session's claim never uses up the day.
	retryDenied = time.Hour
	// RunLimit bounds a whole run, so a detached run never outlives a
	// stalled filesystem.
	RunLimit = 5 * time.Minute
	runEvery = 24 * time.Hour
	// Command is the `sdd-grounding-gate` mode a detached run executes.
	Command = "prune"
	// statusLimit bounds the status read: it holds one run.
	statusLimit = 1 << 20
)

// Status is a run's record. Warned says a SessionStart told its error
// already, so a failure is told once and never again until a run fails anew.
type Status struct {
	Started  string   `json:"started,omitempty"`
	Finished string   `json:"finished,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warned   bool     `json:"warned,omitempty"`
	// Denied counts what the run was not allowed to read or touch; a run with
	// any is claimed again an hour later.
	Denied int `json:"denied,omitempty"`
}

// Starter starts a process that outlives its caller.
type Starter interface {
	Start(ctx context.Context, path string, args ...string) error
}

// ErrRunning is a run another process is making.
var ErrRunning = errors.New("a release prune is running")

func statusPath(dir string) string { return filepath.Join(dir, StatusName) }

func readStatus(dir string) Status {
	var held Status

	file, err := os.Open(statusPath(dir))
	if err != nil {
		return held
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, statusLimit))
	if err != nil {
		return Status{}
	}

	_ = json.Unmarshal(data, &held)

	return held
}

func writeStatus(dir string, held Status) error {
	data, err := json.MarshalIndent(held, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the release prune status: %w", err)
	}

	err = atomicfile.Write(dir, statusPath(dir), data, 0o600)
	if err != nil {
		return fmt.Errorf("writing the release prune status: %w", err)
	}

	return nil
}

// locked runs change on the status under its lock, taken without waiting:
// another process holding it is claiming or recording a run already.
func locked(dir string, change func(held *Status) bool) (bool, error) {
	err := safefs.MkdirAll(dir, dir, 0o700)
	if err != nil {
		return false, fmt.Errorf("creating the data directory: %w", err)
	}

	file, err := safefs.Lock(dir, filepath.Join(dir, lockName), 0o600)
	if err != nil {
		return false, fmt.Errorf("opening the release prune lock: %w", err)
	}

	defer func() { _ = file.Close() }()

	taken, err := filelock.TryExclusive(file)
	if err != nil {
		return false, fmt.Errorf("locking the release prune status: %w", err)
	}

	if !taken {
		return false, nil
	}

	defer func() { _ = filelock.Unlock(file) }()

	held := readStatus(dir)
	if !change(&held) {
		return false, nil
	}

	return true, writeStatus(dir, held)
}

// runLock takes the machine-wide run lock without waiting.
func runLock(dir string) (func(), error) {
	err := safefs.MkdirAll(dir, dir, 0o700)
	if err != nil {
		return nil, fmt.Errorf("creating the data directory: %w", err)
	}

	file, err := safefs.Lock(dir, filepath.Join(dir, runLockName), 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening the release prune run lock: %w", err)
	}

	taken, err := filelock.TryExclusive(file)
	if err != nil || !taken {
		_ = file.Close()

		if err != nil {
			return nil, fmt.Errorf("locking the release prune run: %w", err)
		}

		return nil, ErrRunning
	}

	return func() {
		_ = filelock.Unlock(file)
		_ = file.Close()
	}, nil
}

// Claim takes the day's run: where no run started within the last day, it
// stamps now as the start and answers true, so of every process asking the
// first alone starts one. A caller pays one read of the status for it: the
// lock is taken only on the day a run is due.
func Claim(dir string, now time.Time) (bool, error) {
	due := func(held Status) bool {
		started, err := time.Parse(time.RFC3339, held.Started)

		return err != nil || now.Sub(started) >= runEvery || started.After(now)
	}

	if !due(readStatus(dir)) {
		return false, nil
	}

	return locked(dir, func(held *Status) bool {
		if !due(*held) {
			return false
		}

		held.Started = now.UTC().Format(time.RFC3339)

		return true
	})
}

// Start claims the day's run in the data directory dir and starts
// `<executable> prune`, detached, so its caller never waits for it.
func Start(ctx context.Context, dir string, starter Starter, executable string, now time.Time) error {
	if dir == "" || starter == nil || executable == "" {
		return nil
	}

	claimed, err := Claim(dir, now)
	if err != nil || !claimed {
		return err
	}

	return starter.Start(ctx, executable, Command)
}

// Run deletes every release folder no session may still run
// (pruneReleases), then records the run in the data directory.
func Run(ctx context.Context, releases Releases, now time.Time) Status {
	result := Status{Started: now.UTC().Format(time.RFC3339)}
	if releases.Dir == "" {
		return result
	}

	// One run at a time on the machine: a run started by hand beside the
	// daily one, or a second process claiming across a clock change, exits
	// at once.
	release, err := runLock(releases.Dir)
	if errors.Is(err, ErrRunning) {
		return result
	}

	if errors.Is(err, fs.ErrPermission) {
		result.Denied++

		return result
	}

	if err != nil {
		result.Error = err.Error()

		return result
	}

	defer release()

	deleted, err := pruneReleases(ctx, releases, now)
	result.Deleted = deleted

	switch {
	// A run inside a sandbox may not read the process table or touch the
	// data directory: that is skipped, not failed, and the day's run is
	// retried from a session that may.
	case errors.Is(err, fs.ErrPermission):
		result.Denied++
	case err != nil:
		result.Error = err.Error()
	}

	result.Finished = now.UTC().Format(time.RFC3339)

	_, err = locked(releases.Dir, func(stored *Status) bool {
		started := stored.Started
		*stored = result

		if started != "" {
			stored.Started = started
		}

		if result.Denied > 0 {
			stored.Started = now.Add(retryDenied - runEvery).UTC().Format(time.RFC3339)
		}

		return true
	})
	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}

	return result
}

// Warning is the line a failed run owes, once: empty where the last run
// succeeded or its failure was told already. The status is read without the
// lock first, so a SessionStart after a clean run pays one read.
func Warning(dir string) string {
	line := ""
	if dir == "" {
		return line
	}

	unread := readStatus(dir)
	if unread.Error == "" || unread.Warned {
		return line
	}

	_, _ = locked(dir, func(held *Status) bool {
		if held.Error == "" || held.Warned {
			return false
		}

		held.Warned = true
		line = fmt.Sprintf("release prune failed at %s: %s; see %s", held.Finished, held.Error, statusPath(dir))

		return true
	})

	return line
}
