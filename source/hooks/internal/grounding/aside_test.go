package grounding_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func withVerifyMode(env grounding.Env, mode string) grounding.Env {
	env.SddValues = &test.SddValues{Values: map[string]string{"verify.mode": mode, "sessions.token_saving": "off"}}

	return env
}

func TestTheGateStandsAsideWhereSddGatesGraphWrites(t *testing.T) {
	env := withVerifyMode(bare(t), "prose")

	run(t, env, "start", map[string]any{"session_id": "s1", "source": "startup"})

	for name, payload := range unread(env, "s1") {
		t.Run(name+" passes", func(t *testing.T) {
			got := run(t, env, "gate", payload)

			passes(t, got)

			if got.stderr != "" || got.stdout != "" {
				t.Errorf("a gate standing aside should say nothing, got: %+v", got)
			}
		})
	}

	ground(t, env)

	_, err := os.Stat(filepath.Join(env.ProjectDir, ".sdd", "tmp", "grounding-gate"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a gate standing aside should keep no evidence, got: %v", err)
	}
}

func TestTheGateStillWorksWhereSddsOwnGateIsOff(t *testing.T) {
	env := withVerifyMode(bare(t), "off")

	refuses(t, run(t, env, "gate", ask("s1")))
}

func TestASessionEndingAfterTheGateStoodAsideLeavesNoLedger(t *testing.T) {
	env := withVerifyMode(bare(t), "off")
	ledgers := filepath.Join(env.ProjectDir, ".sdd", "tmp", "grounding-gate")

	record(t, env, as(test.Session, bash(literalRead)), as(test.Session, bash(semanticRead)))

	before, err := filepath.Glob(filepath.Join(ledgers, test.Session+".*"))
	if err != nil || len(before) == 0 {
		t.Fatalf("the reads should be on the ledger before sdd gates graph writes, got %v (%v)", before, err)
	}

	env = withVerifyMode(env, "prose")
	run(t, env, "end", map[string]any{"session_id": test.Session})

	after, err := filepath.Glob(filepath.Join(ledgers, test.Session+".*"))
	if err != nil || len(after) != 0 {
		t.Errorf("the session's end should remove its ledgers even with the gate standing aside, left %v (%v)", after, err)
	}
}

func TestCollapsingStandsAsideOnceSddGatesGraphWrites(t *testing.T) {
	env, transcripts := savingOff(t, bare(t)), t.TempDir()
	env = withVerifyMode(env, "grounding")

	printsWhole(t, run(t, env, "collapse", sddCall(t, transcripts, "show", "show.json", "")))
	printsWhole(t, run(t, env, "collapse", sddCall(t, transcripts, "show", "show.json", "")))
}
