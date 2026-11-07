package pieces_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/pieces"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func TestTheSessionStartNamesOwnBinariesOnThePath(t *testing.T) {
	root := t.TempDir()
	stray := filepath.Join(root, "go", "bin")
	plugin := filepath.Join(root, ".claude", "plugins", "cache", "sdd-helpers", "bin")
	other := filepath.Join(root, "other")

	test.WriteFile(t, filepath.Join(stray, "sdd-grounding-gate"), "")
	test.WriteFile(t, filepath.Join(stray, "sdd-capture"), "")
	test.WriteFile(t, filepath.Join(plugin, "sdd-capture"), "")
	test.WriteFile(t, filepath.Join(other, "sdd"), "")
	test.WriteFile(t, filepath.Join(other, "sdd-other"), "")

	cases := []struct {
		name string
		path []string
		says bool
	}{
		{name: "a stray directory", path: []string{other, stray}, says: true},
		{name: "the same directory twice says it once", path: []string{stray, stray}, says: true},
		{name: "the plugin's own bin", path: []string{plugin, other}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := check(t, pieces.Env{
				Settings: settings(t, ""),
				Getenv: func(key string) string {
					if key == "PATH" {
						return strings.Join(tc.path, string(filepath.ListSeparator))
					}

					return ""
				},
			})

			line := "sdd-helpers: " + stray + " on the PATH holds sdd-capture, sdd-grounding-gate, " +
				"which every project's Bash runs ahead of the plugin; remove them\n"

			count := strings.Count(out, line)
			if tc.says && count != 1 {
				t.Errorf("the line should name the sorted binaries once, got:\n%s", out)
			}

			if !tc.says && strings.Contains(out, "on the PATH holds") {
				t.Errorf("no line should appear for sdd itself, another sdd- program or the plugin's bin, got:\n%s", out)
			}
		})
	}
}
