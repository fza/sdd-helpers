package grounding_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
)

// unread is every call the gate refuses a session carrying no reads, from the
// session given.
func unread(env grounding.Env, session string) map[string]map[string]any {
	return map[string]map[string]any{
		"a question":         ask(session),
		"a draft edit":       as(session, edit(filepath.Join(env.ProjectDir, ".sdd", "tmp", "drafts", "d.md"))),
		"a notebook edit":    as(session, map[string]any{"tool_name": "NotebookEdit", "tool_input": map[string]any{"file_path": filepath.Join(env.ProjectDir, "drafts", "n.ipynb")}}),
		"a capture":          as(session, bash("sdd-capture .sdd/tmp/drafts/d.md")),
		"a plain new":        as(session, bash("sdd new d tac body")),
		"a discarded stream": as(session, bash(shownEntry+" --down 2 --up 1 2>/dev/null")),
		"a shallow show":     as(session, bash(shownEntry)),
	}
}

func TestTheGateSwitchedOff(t *testing.T) {
	env := configured(t, "disabled.yaml")

	for name, payload := range unread(env, "s1") {
		t.Run(name+" passes", func(t *testing.T) {
			got := run(t, env, "gate", payload)

			passes(t, got)

			if got.stderr != "" {
				t.Error("a gate switched off should stay silent")
			}
		})
	}

	t.Run("no ledger is written", func(t *testing.T) {
		run(t, env, "start", map[string]any{"session_id": "s1", "source": "compact"})
		ground(t, env)

		_, err := os.Stat(filepath.Join(env.ProjectDir, ".sdd", "tmp", "grounding-gate"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a gate switched off should keep no evidence, got: %v", err)
		}
	})
}

func TestTheGateSwitchedOn(t *testing.T) {
	refuses(t, run(t, configured(t, "enabled.yaml"), "gate", ask("s1")), "search")
}

func TestASwitchThatIsNoBooleanIsAConfigError(t *testing.T) {
	env := configured(t, "enabled-invalid.yaml")

	refuses(t, run(t, env, "gate", ask("s1")), "Fix `.sdd/grounding.yaml`")
}
