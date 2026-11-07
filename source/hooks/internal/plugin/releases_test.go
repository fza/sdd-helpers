package plugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/test"
)

// A release cannot ship without the fixture of the one before it:
// scripts/release-fixture.go writes one for the version plugin.json names. A
// plugin.json naming no version names no release yet, and no fixture may
// stand for one.
func TestTheFixturesMatchTheNamedRelease(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}

	var plugin struct {
		Version string `json:"version"`
	}

	err = json.Unmarshal(raw, &plugin)
	if err != nil {
		t.Fatal(err)
	}

	versions := test.ReleaseVersions(t)

	if plugin.Version == "" && len(versions) > 0 {
		t.Errorf("plugin.json names no release, so no fixture should exist, got %v", versions)
	}

	if plugin.Version != "" && !slices.Contains(versions, "v"+plugin.Version) {
		t.Errorf("no fixture exists for v%s; run go run scripts/release-fixture.go v%s", plugin.Version, plugin.Version)
	}
}
