package pieces

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// pluginsFolder is where Claude Code installs a plugin; its `bin/` is the one
// place on the PATH an sdd-helpers binary belongs.
const pluginsFolder = "/.claude/plugins/"

// binaries is every program a release ships.
var binaries = []string{"sdd-capture", "sdd-grounding-gate"}

// pathBinaryLines says, one line per directory, which sdd-helpers binaries sit
// on the PATH outside a plugin. Every project's Bash calls find them there
// ahead of the plugin, so an unreleased build runs machine-wide.
func pathBinaryLines(getenv func(string) string) []string {
	if getenv == nil {
		return nil
	}

	var lines []string

	seen := map[string]bool{}

	for _, dir := range filepath.SplitList(getenv("PATH")) {
		if dir == "" || seen[dir] || strings.Contains(dir+string(filepath.Separator), pluginsFolder) {
			continue
		}

		seen[dir] = true

		var names []string

		for _, binary := range binaries {
			_, err := os.Lstat(filepath.Join(dir, binary))
			if err == nil {
				names = append(names, binary)
			}
		}

		if len(names) == 0 {
			continue
		}

		slices.Sort(names)

		lines = append(lines, fmt.Sprintf("%s on the PATH holds %s, which every project's Bash runs ahead of the plugin; remove them",
			dir, strings.Join(names, ", ")))
	}

	return lines
}
