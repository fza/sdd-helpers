package grounding_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/internal/pieces"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func withOptions(t *testing.T, env grounding.Env, options string) grounding.Env {
	t.Helper()

	settings := filepath.Join(t.TempDir(), "settings.json")
	test.WriteFile(t, settings, `{"pluginConfigs":{"sdd-helpers@sdd-helpers":{"options":{`+options+`}}}}`)
	env.Settings = []string{settings}

	return env
}

func TestTheGateOffOnTheMachine(t *testing.T) {
	env := withOptions(t, bare(t), `"grounding_gate_enabled":false`)

	for name, payload := range unread(env, "s1") {
		t.Run(name+" passes", func(t *testing.T) {
			got := run(t, env, "gate", payload)

			passes(t, got)

			if got.stderr != "" {
				t.Error("a gate off on the machine should say nothing")
			}
		})
	}

	run(t, env, "start", map[string]any{"session_id": "s1", "source": "compact"})
	ground(t, env)

	_, err := os.Stat(filepath.Join(env.ProjectDir, ".sdd", "tmp", "grounding-gate"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a gate off on the machine should keep no evidence, got: %v", err)
	}
}

func TestTheCheckSpeaksWithTheGateOffAndNoGraph(t *testing.T) {
	env := withOptions(t, bare(t), `"grounding_gate_enabled":false`)
	env.ProjectDir = t.TempDir()
	env.WorkingDir = env.ProjectDir

	got := runRaw(t, env, "check", "")
	if got.code != 0 || got.stdout != "sdd-helpers: "+pieces.DeprecatedLine+"\n"+
		"sdd-helpers: grounding gate is off, so sdd-capture is off; turn grounding_gate_enabled on to restore it\n" {
		t.Errorf("the check should name a broken tie whatever the gate and the project, got: %+v", got)
	}
}
