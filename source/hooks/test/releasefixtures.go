package test

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// everyFixture keeps the dotted files and folders a release fixture's
// checkout holds, its `.sdd/` among them, which `fixtures` leaves out.
//
//go:embed all:fixtures
var everyFixture embed.FS

const (
	releasesRoot = "fixtures/releases"
	workMark     = "@WORK@"
)

// ReleaseCall is one gate call scripts/release-fixture recorded: its mode,
// the payload it was handed and what the release answered.
type ReleaseCall struct {
	Args   []string `json:"args"`
	Stdin  string   `json:"stdin"`
	Code   int      `json:"code"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
}

// ReleaseDump is a release fixture's dump.json.
type ReleaseDump struct {
	Release string        `json:"release"`
	At      time.Time     `json:"at"`
	Calls   []ReleaseCall `json:"calls"`
}

// ReleaseVersions is every release with a committed fixture. Before the first
// release no fixture folder exists, and there is none.
func ReleaseVersions(t *testing.T) []string {
	t.Helper()

	entries, err := fs.ReadDir(everyFixture, releasesRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		t.Fatalf("listing the release fixtures: %v", err)
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		versions = append(versions, entry.Name())
	}

	return versions
}

// ReleaseDumpOf is the dump.json of the release's fixture, every path the
// release ran in written as root.
func ReleaseDumpOf(t *testing.T, version string, root string) ReleaseDump {
	t.Helper()

	data, err := everyFixture.ReadFile(releasesRoot + "/" + version + "/dump.json")
	if err != nil {
		t.Fatalf("reading the dump of %s: %v", version, err)
	}

	var dump ReleaseDump

	err = json.Unmarshal(data, &dump)
	if err != nil {
		t.Fatalf("parsing the dump of %s: %v", version, err)
	}

	for index := range dump.Calls {
		call := &dump.Calls[index]
		call.Stdin = strings.ReplaceAll(call.Stdin, workMark, root)
		call.Stdout = strings.ReplaceAll(call.Stdout, workMark, root)
		call.Stderr = strings.ReplaceAll(call.Stderr, workMark, root)
	}

	return dump
}

// ReleaseTree lays the release's checkout into root/repo as the script's
// session left it, root standing wherever a file names the folder the
// release ran in.
func ReleaseTree(t *testing.T, version string, root string) {
	t.Helper()

	base := releasesRoot + "/" + version + "/checkout"

	err := fs.WalkDir(everyFixture, base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		content, err := everyFixture.ReadFile(path)
		if err != nil {
			return err
		}

		name := strings.TrimPrefix(path, base+"/")
		WriteFile(t, filepath.Join(root, "repo", filepath.FromSlash(name)),
			strings.ReplaceAll(string(content), workMark, root))

		return nil
	})
	if err != nil {
		t.Fatalf("laying the checkout of %s: %v", version, err)
	}
}
