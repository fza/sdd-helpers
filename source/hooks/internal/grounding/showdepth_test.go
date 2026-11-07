package grounding_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

const shownEntry = "sdd show 20260907-164635-d-tac-vkm"

// configured is a project carrying a graph and the named grounding.yaml
// fixture.
func configured(t *testing.T, fixture string) grounding.Env {
	t.Helper()

	env := bare(t)
	test.WriteFile(t, filepath.Join(env.ProjectDir, ".sdd", "grounding.yaml"), test.Fixture(t, "grounding/"+fixture))

	return env
}

func as(session string, payload map[string]any) map[string]any {
	return with(payload, "session_id", session)
}

func TestConfiguredShowDepth(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		command string
		says    string
	}{
		{name: "a deeper down refuses the default", fixture: "show-depth-down.yaml", command: " --down 2 --up 1", says: "`--down 2` reads short of the depth owed; rerun as:"},
		{name: "the refusal reruns at the depth owed", fixture: "show-depth-down.yaml", command: " --down 2 --up 1", says: "rerun as:\n\n  " + shownEntry + " --down 3 --up 1\n"},
		{name: "both directions set", fixture: "show-depth.yaml", command: " --down 3 --up 1", says: "`--up 1` reads short of the depth owed; rerun as:\n\n  " + shownEntry + " --down 3 --up 2\n"},
		{name: "the default stands without a block", fixture: "area-prefix.yaml", command: " --down 1 --up 1", says: "  " + shownEntry + " --down 2 --up 1\n"},
		{name: "the default stands under an empty block", fixture: "show-depth-empty.yaml", command: "", says: "  " + shownEntry + " --down 2 --up 1\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refuses(t, run(t, configured(t, tc.fixture), "gate", bash(shownEntry+tc.command)), tc.says)
		})
	}

	passing := []struct {
		name    string
		fixture string
		command string
	}{
		{name: "the depth owed", fixture: "show-depth-down.yaml", command: " --down 3 --up 1"},
		{name: "deeper than owed", fixture: "show-depth.yaml", command: " --down 4 --up 2"},
		{name: "nothing owed", fixture: "show-depth-none.yaml"},
	}

	for _, tc := range passing {
		t.Run(tc.name, func(t *testing.T) {
			passes(t, run(t, configured(t, tc.fixture), "gate", bash(shownEntry+tc.command)))
		})
	}
}

func TestANegativeDepthIsAConfigError(t *testing.T) {
	env := configured(t, "show-depth-negative.yaml")

	refuses(t, run(t, env, "gate", ask("s1")), "below 0")
	refuses(t, run(t, env, "gate", bash(shownEntry+" --down 2 --up 1")), "below 0")

	passes(t, run(t, env, "gate", bash("ls")))
}

func TestTheShowRefusalNamesTheDepthOnce(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		fixture string
		depth   string
	}{
		{name: "a Bash show", payload: bash(shownEntry + " --down 1 | head -40"), fixture: "refusals/downstream.txt", depth: "--down 2"},
		{name: "an MCP show", payload: mcp("show", map[string]any{"ids": []string{"20260907-164635-d-tac-vkm"}}),
			fixture: "refusals/downstream-mcp.txt", depth: `"down": 2`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, bare(t), "gate", tc.payload)
			if got.said != test.Fixture(t, tc.fixture) {
				t.Errorf("the refusal should echo the call at the depth owed above the rule, got: %s", got.said)
			}

			if strings.Count(got.said, tc.depth) != 1 {
				t.Errorf("the refusal should name the depth owed only in the call it echoes, got: %s", got.said)
			}
		})
	}
}

func TestTheShowRefusalCorrectsOnlyWhatFellShort(t *testing.T) {
	cases := []struct {
		name    string
		command string
		rerun   string
	}{
		{name: "a deeper direction stays as given", command: " --down 5 --up 0", rerun: shownEntry + " --down 5 --up 1"},
		{name: "an equals sign", command: " --down=0 --up 1", rerun: shownEntry + " --down 2 --up 1"},
		{name: "a missing direction is appended", command: " --up 3", rerun: shownEntry + " --up 3 --down 2"},
		{name: "a subject flag stays", command: " --down 1 --entry e", rerun: shownEntry + " --down 2 --entry e --up 1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refuses(t, run(t, bare(t), "gate", bash(shownEntry+tc.command)), "rerun as:\n\n  "+tc.rerun+"\n\n")
		})
	}
}
