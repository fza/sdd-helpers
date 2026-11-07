package grounding_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

// underPlugin is env run as the plugin's hook: its variables name a data
// directory and a plugin root naming version 1.4.0, and the starts it makes
// are recorded.
func underPlugin(t *testing.T, env grounding.Env) (grounding.Env, string, *test.Starts) {
	t.Helper()

	root := t.TempDir()
	data := filepath.Join(t.TempDir(), "data")
	test.WriteFile(t, filepath.Join(root, ".claude-plugin", "plugin.json"), `{"name": "sdd-helpers", "version": "1.4.0"}`)
	test.WriteFile(t, filepath.Join(data, "1.4.0", "sdd-grounding-gate"), "#!/bin/sh\n")

	variables := map[string]string{"CLAUDE_PLUGIN_ROOT": root, "CLAUDE_PLUGIN_DATA": data}
	starts := &test.Starts{}

	env.Getenv = func(key string) string { return variables[key] }
	env.Starter = starts
	env.Executable = filepath.Join(data, "1.4.0", "sdd-grounding-gate")
	env.Processes = test.Executables{}

	return env, data, starts
}

func TestASessionStartStartsTheDaysPruneOnce(t *testing.T) {
	env, _, starts := underPlugin(t, bare(t))

	for range 2 {
		got := run(t, env, "start", map[string]any{"session_id": test.Session, "source": "startup"})
		if got.code != 0 || got.stdout != "" || got.stderr != "" {
			t.Fatalf("a session start should pass silently, got %+v", got)
		}
	}

	started := starts.Started()
	if len(started) != 1 || !slices.Equal(started[0], []string{env.Executable, "prune"}) {
		t.Errorf("the first session start of the day alone should start this binary's prune, got %v", started)
	}
}

func TestAFailedPruneIsToldAtTheNextSessionStartOnce(t *testing.T) {
	env, data, _ := underPlugin(t, bare(t))
	env.Processes = test.Executables{Err: os.ErrInvalid}

	pruned := run(t, env, "prune", map[string]any{})
	if pruned.code != 1 || !strings.HasPrefix(pruned.stderr, "sdd-helpers: reading the process table") {
		t.Fatalf("a failing prune should exit 1 naming its error, got %+v", pruned)
	}

	first := run(t, env, "start", map[string]any{"session_id": test.Session, "source": "startup"})
	want := "sdd-helpers: release prune failed at "
	if first.code != 0 || !strings.HasPrefix(first.stdout, want) ||
		!strings.HasSuffix(first.stdout, "; see "+filepath.Join(data, "release-prune.json")+"\n") {
		t.Errorf("the next session start should tell the failure, naming the status file, got %+v", first)
	}

	second := run(t, env, "start", map[string]any{"session_id": test.Session, "source": "startup"})
	if second.code != 0 || second.stdout != "" {
		t.Errorf("a failure should be told once, got %+v", second)
	}
}

func TestPruneDeletesTheReleasesNoSessionMayRun(t *testing.T) {
	env, data, _ := underPlugin(t, bare(t))
	test.WriteFile(t, filepath.Join(data, "1.0.0", "sdd-grounding-gate"), "#!/bin/sh\n")

	got := run(t, env, "prune", map[string]any{})
	if got.code != 0 || got.stdout != "deleted "+filepath.Join(data, "1.0.0")+"\n" || got.stderr != "" {
		t.Errorf("prune should delete and name each release no session may run, got %+v", got)
	}

	if !test.Exists(t, filepath.Join(data, "1.4.0", "sdd-grounding-gate")) {
		t.Error("the named release should stay")
	}
}

func TestPruneRunsInAProjectWithoutAGraph(t *testing.T) {
	root := t.TempDir()
	env, data, _ := underPlugin(t, grounding.Env{ProjectDir: root, WorkingDir: root, Runner: &test.Runner{}})
	test.WriteFile(t, filepath.Join(data, "1.0.0", "sdd-grounding-gate"), "#!/bin/sh\n")

	got := run(t, env, "prune", map[string]any{})
	if got.code != 0 || test.Exists(t, filepath.Join(data, "1.0.0")) {
		t.Errorf("prune works on the machine, not the project, got %+v", got)
	}
}

func TestPruneRunByHandTouchesNothing(t *testing.T) {
	env, data, _ := underPlugin(t, bare(t))
	env.Getenv = func(string) string { return "" }
	test.WriteFile(t, filepath.Join(data, "1.0.0", "sdd-grounding-gate"), "#!/bin/sh\n")

	got := run(t, env, "prune", map[string]any{})
	if got.code != 0 || got.stdout != "" || !test.Exists(t, filepath.Join(data, "1.0.0")) {
		t.Errorf("prune without the plugin's variables should touch nothing, got %+v", got)
	}
}

func TestTheMachineCheckStartsTheDaysPruneInAProjectWithoutAGraph(t *testing.T) {
	root := t.TempDir()
	env, _, starts := underPlugin(t, grounding.Env{ProjectDir: root, WorkingDir: root, Runner: &test.Runner{}})

	for range 2 {
		got := run(t, env, "check", map[string]any{"session_id": test.Session})
		if got.code != 0 {
			t.Fatalf("the machine check should pass, got %+v", got)
		}
	}

	started := starts.Started()
	if len(started) != 1 || !slices.Equal(started[0], []string{env.Executable, "prune"}) {
		t.Errorf("the machine check should start the day's prune once, even without a graph, got %v", started)
	}
}

func TestASessionStartWithoutAGraphStartsNoPrune(t *testing.T) {
	root := t.TempDir()
	env, data, starts := underPlugin(t, grounding.Env{ProjectDir: root, WorkingDir: root, Runner: &test.Runner{}})

	got := run(t, env, "start", map[string]any{"session_id": test.Session})
	if got.code != 0 || got.stdout != "" || len(starts.Started()) != 0 ||
		test.Exists(t, filepath.Join(data, "release-prune.json")) {
		t.Errorf("a session start in a project without a graph should start and write nothing, got %+v", got)
	}
}
