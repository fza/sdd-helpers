// Package pluginconfig reads the plugin's options in layers, lowest first: the
// built-in defaults, the answers a user gave Claude Code's `userConfig`
// dialog, the hook environment's CLAUDE_PLUGIN_OPTION_<KEY> variables, then
// Claude Code's managed settings, which win. Claude Code exports the answers
// only to hook processes, so a command the Bash tool runs from the plugin's
// `bin/` reads them from the settings files themselves.
package pluginconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const PluginID = "sdd-helpers@sdd-helpers"

// Options switches each program on or off for the machine; every one is on
// unless set. CaptureRemoveDraft has sdd-capture delete a draft once its
// entry is captured.
type Options struct {
	GroundingGateEnabled bool
	CaptureEnabled       bool
	CaptureRemoveDraft   bool
}

// Keys is every option the plugin reads, as a settings file and the hook
// environment spell it.
var Keys = []string{"grounding_gate_enabled", "capture_enabled", "capture_remove_draft"}

type settings struct {
	PluginConfigs map[string]struct {
		Options map[string]json.RawMessage `json:"options"`
	} `json:"pluginConfigs"`
}

// Paths is every file that can set an option, lowest precedence first: the
// user's settings, under CLAUDE_CONFIG_DIR where it is set, then the managed
// settings and their `managed-settings.d/` drop-ins in name order, as Claude
// Code merges them. Project settings are never among them: a project's
// committed files must not switch a guard off for the machine.
func Paths(home string, goos string, getenv func(string) string) []string {
	config := filepath.Join(home, ".claude")
	if getenv != nil && getenv("CLAUDE_CONFIG_DIR") != "" {
		config = getenv("CLAUDE_CONFIG_DIR")
	}

	managed := map[string]string{
		"darwin": "/Library/Application Support/ClaudeCode", "linux": "/etc/claude-code",
	}[goos]

	return pathsUnder(config, managed)
}

func pathsUnder(config string, managed string) []string {
	paths := []string{filepath.Join(config, "settings.json")}

	if managed == "" {
		return paths
	}

	paths = append(paths, filepath.Join(managed, "managed-settings.json"))

	dropIns, err := filepath.Glob(filepath.Join(managed, "managed-settings.d", "*.json"))
	if err == nil {
		paths = append(paths, dropIns...)
	}

	return paths
}

// ReadWith folds paths as Paths orders them: the user's settings, then the
// hook environment's CLAUDE_PLUGIN_OPTION_<KEY> variables, then the remaining,
// managed settings, which no environment overrides: they are how a machine
// locks a program on. A missing file contributes nothing; an unreadable file,
// or a value that is no boolean, is an error. An empty value is unset.
func ReadWith(paths []string, getenv func(string) string) (Options, error) {
	options := Defaults()

	user, managed := paths[:min(1, len(paths))], paths[min(1, len(paths)):]

	err := options.foldSettings(user)
	if err != nil {
		return Options{}, err
	}

	if getenv != nil {
		values := map[string]json.RawMessage{}

		for _, key := range Keys {
			raw, err := json.Marshal(getenv("CLAUDE_PLUGIN_OPTION_" + strings.ToUpper(key)))
			if err != nil {
				return Options{}, fmt.Errorf("encoding %s: %w", key, err)
			}

			values[key] = raw
		}

		err = options.apply(values)
		if err != nil {
			return Options{}, fmt.Errorf("reading the hook environment's options: %w", err)
		}
	}

	err = options.foldSettings(managed)
	if err != nil {
		return Options{}, err
	}

	return options, nil
}

func (options *Options) foldSettings(paths []string) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var parsed settings

		err = json.Unmarshal(raw, &parsed)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		err = options.apply(parsed.PluginConfigs[PluginID].Options)
		if err != nil {
			return fmt.Errorf("reading the %s options in %s: %w", PluginID, path, err)
		}
	}

	return nil
}

func Defaults() Options {
	return Options{GroundingGateEnabled: true, CaptureEnabled: true, CaptureRemoveDraft: true}
}

func (options *Options) switches() map[string]*bool {
	return map[string]*bool{
		"grounding_gate_enabled": &options.GroundingGateEnabled,
		"capture_enabled":        &options.CaptureEnabled,
		"capture_remove_draft":   &options.CaptureRemoveDraft,
	}
}

// apply sets every key values holds, all or none.
func (options *Options) apply(values map[string]json.RawMessage) error {
	changed := *options

	for key, target := range changed.switches() {
		raw, found := values[key]
		if !found || unset(raw) {
			continue
		}

		value, err := boolean(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}

		*target = value
	}

	*options = changed

	return nil
}

func unset(raw json.RawMessage) bool {
	text := strings.TrimSpace(string(raw))

	return text == "" || text == "null" || text == `""`
}

// boolean accepts a JSON boolean or the string the dialog may store for one.
func boolean(raw json.RawMessage) (bool, error) {
	var value bool

	err := json.Unmarshal(raw, &value)
	if err == nil {
		return value, nil
	}

	var text string

	err = json.Unmarshal(raw, &text)
	if err == nil && (text == "true" || text == "false") {
		return text == "true", nil
	}

	return false, fmt.Errorf("%s is not a boolean", raw)
}
