package releaseprune_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/fza/sdd-helpers/source/hooks/internal/releaseprune"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

const named = "1.4.0"

var now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

// placeRelease unpacks a fake release of version into dir, its use mark
// written at used, or none where used is zero.
func placeRelease(t *testing.T, dir string, version string, used time.Time) {
	t.Helper()

	test.WriteFile(t, filepath.Join(dir, version, "sdd-grounding-gate"), "#!/bin/sh\n")

	if used.IsZero() {
		return
	}

	mark := filepath.Join(dir, version, releaseprune.UsedName)
	test.WriteFile(t, mark, "\n")

	err := os.Chtimes(mark, used, used)
	if err != nil {
		t.Fatalf("dating the use mark of %s: %v", version, err)
	}
}

func TestARunDeletesOnlyTheReleasesNoSessionMayRun(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	link := filepath.Join(t.TempDir(), "data-link")

	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		t.Fatalf("creating the data directory: %v", err)
	}

	err = os.Symlink(dir, link)
	if err != nil {
		t.Fatalf("linking the data directory: %v", err)
	}

	cases := []struct {
		name    string
		version string
		used    time.Time
		deleted bool
	}{
		{name: "the named version stays, even unused", version: named},
		{name: "a version a process runs from stays", version: "1.2.0"},
		{name: "a version a process runs from through the resolved directory stays", version: "1.1.0"},
		{name: "a version used within two weeks stays", version: "1.3.0", used: now.Add(-13 * 24 * time.Hour)},
		{name: "a version used over two weeks ago goes", version: "1.0.0", used: now.Add(-15 * 24 * time.Hour),
			deleted: true},
		{name: "a version never marked goes", version: "0.9.0", deleted: true},
	}

	for _, tc := range cases {
		placeRelease(t, dir, tc.version, tc.used)
	}

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving the data directory: %v", err)
	}

	processes := test.Executables{Paths: []string{
		filepath.Join(link, "1.2.0", "sdd-grounding-gate"),
		filepath.Join(resolved, "1.1.0", "sdd-capture"),
		"/usr/local/bin/sdd-capture",
	}}

	result := releaseprune.Run(context.Background(),
		releaseprune.Releases{Dir: link, Named: named, Processes: processes}, now)
	if result.Error != "" {
		t.Fatalf("the run should succeed: %s", result.Error)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if test.Exists(t, filepath.Join(dir, tc.version)) == tc.deleted {
				t.Error("a release should go exactly when no session may still run it")
			}

			if slices.Contains(result.Deleted, filepath.Join(link, tc.version)) != tc.deleted {
				t.Error("the run should record exactly the releases it deleted")
			}
		})
	}
}

func TestARunLeavesEverythingButReleaseFolders(t *testing.T) {
	dir := t.TempDir()

	placeRelease(t, dir, "1.0.0", time.Time{})
	test.WriteFile(t, filepath.Join(dir, "notes", "1.0.0"), "kept\n")
	test.WriteFile(t, filepath.Join(dir, "1.0", "sdd-capture"), "#!/bin/sh\n")
	test.WriteFile(t, filepath.Join(dir, "v1.0.0", "sdd-capture"), "#!/bin/sh\n")
	test.WriteFile(t, filepath.Join(dir, ".fetch-1.5.0.4242", "checksums.txt"), "sum\n")
	test.WriteFile(t, filepath.Join(dir, "1.1.0"), "a file named as a version\n")
	test.WriteFile(t, filepath.Join(dir, "README"), "kept by Claude Code\n")

	outside := t.TempDir()
	placeRelease(t, outside, "0.9.0", time.Time{})

	err := os.Symlink(filepath.Join(outside, "0.9.0"), filepath.Join(dir, "0.9.0"))
	if err != nil {
		t.Fatalf("linking a release: %v", err)
	}

	result := releaseprune.Run(context.Background(),
		releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)
	if result.Error != "" {
		t.Fatalf("the run should succeed: %s", result.Error)
	}

	if test.Exists(t, filepath.Join(dir, "1.0.0")) {
		t.Error("an unused release folder should go")
	}

	kept := []string{"notes", "1.0", "v1.0.0", ".fetch-1.5.0.4242", "1.1.0", "README", "0.9.0",
		releaseprune.StatusName}
	for _, name := range kept {
		_, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("anything but a folder named as a version should stay, lost %s", name)
		}
	}

	if !test.Exists(t, filepath.Join(outside, "0.9.0", "sdd-grounding-gate")) {
		t.Error("a link named as a version should never be followed")
	}
}

func TestARunRemovesAReleaseACrashLeftMovedAside(t *testing.T) {
	dir := t.TempDir()

	test.WriteFile(t, filepath.Join(dir, ".pruned-1.0.0", "sdd-grounding-gate"), "#!/bin/sh\n")
	test.WriteFile(t, filepath.Join(dir, ".pruned-notes", "kept"), "kept\n")

	result := releaseprune.Run(context.Background(),
		releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)
	if result.Error != "" {
		t.Fatalf("the run should succeed: %s", result.Error)
	}

	if test.Exists(t, filepath.Join(dir, ".pruned-1.0.0")) || len(result.Deleted) > 0 {
		t.Error("a release a crash left moved aside should go with the next run, never counted as a release")
	}

	if !test.Exists(t, filepath.Join(dir, ".pruned-notes")) {
		t.Error("a moved-aside folder not named after a version should stay")
	}
}

func TestARunThatCannotTellWhatRunsKeepsEveryRelease(t *testing.T) {
	cases := []struct {
		name      string
		releases  func(dir string) releaseprune.Releases
		denied    bool
		withError bool
	}{
		{name: "no named version, as a plugin naming no release", releases: func(dir string) releaseprune.Releases {
			return releaseprune.Releases{Dir: dir, Processes: test.Executables{}}
		}},
		{name: "no process table", releases: func(dir string) releaseprune.Releases {
			return releaseprune.Releases{Dir: dir, Named: named}
		}},
		{name: "a process table denied, as in a sandbox", denied: true, releases: func(dir string) releaseprune.Releases {
			return releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{Err: os.ErrPermission}}
		}},
		{name: "a process table that fails", withError: true, releases: func(dir string) releaseprune.Releases {
			return releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{Err: os.ErrInvalid}}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			placeRelease(t, dir, "1.0.0", time.Time{})

			result := releaseprune.Run(context.Background(), tc.releases(dir), now)

			if !test.Exists(t, filepath.Join(dir, "1.0.0")) || len(result.Deleted) > 0 {
				t.Error("a run that cannot tell which releases a session runs should keep every one")
			}

			if (result.Denied > 0) != tc.denied || (result.Error != "") != tc.withError {
				t.Errorf("a denied process table should count as denied and a failing one as an error, got %+v", result)
			}
		})
	}
}

func TestARunPastItsLimitDeletesNoMoreReleases(t *testing.T) {
	dir := t.TempDir()
	placeRelease(t, dir, "1.0.0", time.Time{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := releaseprune.Run(ctx, releaseprune.Releases{Dir: dir, Named: named, Processes: test.Executables{}}, now)

	if !test.Exists(t, filepath.Join(dir, "1.0.0")) || result.Error == "" {
		t.Error("a run stopped by its limit should delete no further release and record why it stopped")
	}
}

func TestThePluginsOwnVariablesNameItsReleases(t *testing.T) {
	root := t.TempDir()
	data := t.TempDir()
	test.WriteFile(t, filepath.Join(root, ".claude-plugin", "plugin.json"),
		"{\n  \"name\": \"sdd-helpers\",\n  \"version\": \""+named+"\"\n}\n")

	processes := test.Executables{}

	cases := []struct {
		name      string
		variables map[string]string
		want      releaseprune.Releases
	}{
		{name: "a hook's environment",
			variables: map[string]string{"CLAUDE_PLUGIN_ROOT": root, "CLAUDE_PLUGIN_DATA": data},
			want:      releaseprune.Releases{Dir: data, Named: named}},
		{name: "a command run by hand", variables: map[string]string{}},
		{name: "no plugin root", variables: map[string]string{"CLAUDE_PLUGIN_DATA": data}},
		{name: "a relative data directory",
			variables: map[string]string{"CLAUDE_PLUGIN_ROOT": root, "CLAUDE_PLUGIN_DATA": "data"}},
		{name: "a plugin root without a manifest",
			variables: map[string]string{"CLAUDE_PLUGIN_ROOT": t.TempDir(), "CLAUDE_PLUGIN_DATA": data},
			want:      releaseprune.Releases{Dir: data}},
	}

	if releaseprune.PluginReleases(nil, processes).Dir != "" {
		t.Error("no environment should name no releases")
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := releaseprune.PluginReleases(func(key string) string { return tc.variables[key] }, processes)
			if got.Dir != tc.want.Dir || got.Named != tc.want.Named {
				t.Errorf("the releases should come from the plugin's own variables and manifest alone, got %+v", got)
			}
		})
	}
}
