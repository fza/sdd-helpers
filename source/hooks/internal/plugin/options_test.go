package plugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
)

type declaredOption struct {
	Type    string          `json:"type"`
	Default json.RawMessage `json:"default"`
}

func userConfig(t *testing.T) map[string]declaredOption {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("reading plugin.json: %v", err)
	}

	var manifest struct {
		UserConfig map[string]declaredOption `json:"userConfig"`
	}

	err = json.Unmarshal(raw, &manifest)
	if err != nil {
		t.Fatalf("parsing plugin.json: %v", err)
	}

	return manifest.UserConfig
}

func TestTheReadmeListsEveryOption(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(repository, "README.md"))
	if err != nil {
		t.Fatalf("reading the README: %v", err)
	}

	_, after, found := strings.Cut(string(readme), "\n## Options\n")
	section, _, _ := strings.Cut(after, "\n## ")

	if !found {
		t.Fatal("the README should carry an Options section")
	}

	for _, key := range pluginconfig.Keys {
		if !strings.Contains(section, "`"+key+"`") {
			t.Errorf("the README's Options section should name the option %s", key)
		}
	}

	if !strings.Contains(string(readme), "\n## Dependencies\n") {
		t.Error("the README should carry a Dependencies section")
	}
}

func TestEveryDeclaredOptionIsRead(t *testing.T) {
	var declared []string
	for key := range userConfig(t) {
		declared = append(declared, key)
	}

	slices.Sort(declared)

	read := slices.Sorted(slices.Values(pluginconfig.Keys))
	if !slices.Equal(read, declared) {
		t.Errorf("the plugin should declare every option it reads and no other, read %v, declared %v", read, declared)
	}
}

// Claude Code's dialog saved untouched stores every boolean: false for one
// declaring no default. Each switch therefore declares the reader's built-in
// default, so an untouched save switches nothing off.
func TestEverySwitchDeclaresTheReadersDefault(t *testing.T) {
	for key, declared := range userConfig(t) {
		t.Run(key, func(t *testing.T) {
			if declared.Type != "boolean" {
				t.Fatalf("every option should be a boolean switch, got %s", declared.Type)
			}

			options, err := pluginconfig.ReadWith(nil, func(name string) string {
				if name == "CLAUDE_PLUGIN_OPTION_"+strings.ToUpper(key) {
					return string(declared.Default)
				}

				return ""
			})
			if err != nil || options != pluginconfig.Defaults() {
				t.Errorf("the declared default should be the reader's built-in one, got %s: %+v %v", declared.Default, options, err)
			}
		})
	}
}
