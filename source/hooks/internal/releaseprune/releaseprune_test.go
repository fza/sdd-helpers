package releaseprune_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fza/sdd-helpers/source/hooks/internal/filelock"
	"github.com/fza/sdd-helpers/source/hooks/internal/releaseprune"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

// failing is a run whose process table fails: an error that is no denial.
func failing(dir string) releaseprune.Releases {
	return releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{Err: os.ErrInvalid}}
}

func TestARunExitsWhileAnotherRuns(t *testing.T) {
	dir := t.TempDir()
	placeRelease(t, dir, "1.0.0", time.Time{})

	lock, err := os.OpenFile(filepath.Join(dir, "release-prune.run.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("opening the run lock: %v", err)
	}

	defer func() { _ = lock.Close() }()

	taken, err := filelock.TryExclusive(lock)
	if err != nil || !taken {
		t.Fatalf("taking the run lock: %v", err)
	}

	result := releaseprune.Run(context.Background(),
		releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)
	if result.Error != "" || len(result.Deleted) > 0 || !test.Exists(t, filepath.Join(dir, "1.0.0")) {
		t.Error("a run should exit at once while another holds the run lock")
	}
}

func TestARunRecordsItsOutcome(t *testing.T) {
	dir := t.TempDir()
	placeRelease(t, dir, "1.0.0", time.Time{})

	releaseprune.Run(context.Background(), releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)

	data, err := os.ReadFile(filepath.Join(dir, releaseprune.StatusName))
	if err != nil {
		t.Fatalf("reading the status: %v", err)
	}

	var status releaseprune.Status

	err = json.Unmarshal(data, &status)
	if err != nil {
		t.Fatalf("decoding the status: %v", err)
	}

	if status.Finished == "" || !slices.Equal(status.Deleted, []string{filepath.Join(dir, "1.0.0")}) || status.Error != "" {
		t.Error("the status should name when the run finished and what it deleted")
	}
}

func TestARunWithoutADataDirectoryTouchesNothing(t *testing.T) {
	result := releaseprune.Run(context.Background(), releaseprune.Releases{}, now)
	if result.Error != "" || result.Finished != "" || len(result.Deleted) > 0 {
		t.Errorf("a run naming no data directory should do nothing, got %+v", result)
	}
}

func TestTheDaysClaimStartsOneRun(t *testing.T) {
	dir := t.TempDir()
	starts := &test.Starts{}
	executable := filepath.Join(t.TempDir(), "1.4.0", "sdd-grounding-gate")

	for _, at := range []time.Time{now, now.Add(time.Hour), now.Add(23 * time.Hour), now.Add(25 * time.Hour)} {
		err := releaseprune.Start(context.Background(), dir, starts, executable, at)
		if err != nil {
			t.Fatalf("starting the run: %v", err)
		}
	}

	started := starts.Started()
	if len(started) != 2 {
		t.Fatalf("one run should start per day, started %v", started)
	}

	if !slices.Equal(started[0], []string{executable, "prune"}) {
		t.Error("the run should be the calling binary's prune mode")
	}
}

func TestStartWithoutWhatARunNeedsStartsNone(t *testing.T) {
	starts := &test.Starts{}

	err := releaseprune.Start(context.Background(), "", starts, "/bin/sdd-grounding-gate", now)
	if err != nil || len(starts.Started()) != 0 {
		t.Error("no data directory should start no run")
	}

	dir := t.TempDir()

	err = releaseprune.Start(context.Background(), dir, starts, "", now)
	if err != nil || len(starts.Started()) != 0 || test.Exists(t, filepath.Join(dir, releaseprune.StatusName)) {
		t.Error("no executable should start no run and claim no day")
	}
}

func TestAFailedRunIsWarnedOnce(t *testing.T) {
	dir := t.TempDir()

	result := releaseprune.Run(context.Background(), failing(dir), now)
	if result.Error == "" {
		t.Fatal("a process table that fails should fail the run")
	}

	first := releaseprune.Warning(dir)
	if !strings.Contains(first, filepath.Join(dir, releaseprune.StatusName)) || !strings.Contains(first, result.Error) {
		t.Errorf("the first warning should name the error and the status file, got %q", first)
	}

	if releaseprune.Warning(dir) != "" {
		t.Error("a failure should be warned once")
	}

	releaseprune.Run(context.Background(), failing(dir), now.Add(48*time.Hour))

	if releaseprune.Warning(dir) == "" {
		t.Error("a new failure should be warned again")
	}
}

func TestASucceedingRunIsNeverWarned(t *testing.T) {
	dir := t.TempDir()

	releaseprune.Run(context.Background(), releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)

	if releaseprune.Warning(dir) != "" || releaseprune.Warning("") != "" {
		t.Error("a run without an error, or no data directory, should warn nothing")
	}
}

func TestTheChecksOnAQuietDayTakeNoLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	starts := &test.Starts{}
	executable := filepath.Join(t.TempDir(), "sdd-grounding-gate")

	if releaseprune.Warning(dir) != "" || test.Exists(t, dir) {
		t.Error("a warning check with no run recorded should create nothing")
	}

	err := releaseprune.Start(context.Background(), dir, starts, executable, now)
	if err != nil {
		t.Fatalf("starting the run: %v", err)
	}

	lock := filepath.Join(dir, "release-prune.lock")

	err = os.Remove(lock)
	if err != nil {
		t.Fatalf("removing the status lock: %v", err)
	}

	err = releaseprune.Start(context.Background(), dir, starts, executable, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("checking the run: %v", err)
	}

	_ = releaseprune.Warning(dir)

	if test.Exists(t, lock) {
		t.Error("the claim and the warning check should only read the status on a day the run started already")
	}
}

func TestARunDeniedIsSkippedAndRetried(t *testing.T) {
	dir := t.TempDir()
	placeRelease(t, dir, "1.0.0", time.Time{})

	claimed, err := releaseprune.Claim(dir, now)
	if err != nil || !claimed {
		t.Fatalf("claiming the run: %v", err)
	}

	denied := releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{Err: os.ErrPermission}}

	result := releaseprune.Run(context.Background(), denied, now)
	if result.Error != "" || result.Denied != 1 || !test.Exists(t, filepath.Join(dir, "1.0.0")) {
		t.Fatal("a process table the run may not read should be skipped, not failed")
	}

	if releaseprune.Warning(dir) != "" {
		t.Error("a denied run should warn nothing")
	}

	soon, err := releaseprune.Claim(dir, now.Add(30*time.Minute))
	if err != nil || soon {
		t.Error("a denied run should not be claimed again within the hour")
	}

	later, err := releaseprune.Claim(dir, now.Add(2*time.Hour))
	if err != nil || !later {
		t.Error("a denied run should leave the day's run to a later session")
	}
}

func TestARunDeniedTheDataDirectoryIsSkipped(t *testing.T) {
	dir := t.TempDir()
	placeRelease(t, dir, "1.0.0", time.Time{})

	err := os.Chmod(dir, 0o500)
	if err != nil {
		t.Fatalf("denying the data directory: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	result := releaseprune.Run(context.Background(),
		releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)
	if result.Error != "" || result.Denied != 1 || !test.Exists(t, filepath.Join(dir, "1.0.0")) {
		t.Errorf("a data directory the run may not write should be skipped, not failed, got %+v", result)
	}
}
