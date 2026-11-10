package pieces_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/pieces"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func TestSddGatesWhereItReportsAVerifyModeOtherThanOff(t *testing.T) {
	cases := []struct {
		name   string
		mode   string
		values bool
		gates  bool
	}{
		{name: "the fork's default", mode: "prose", values: true, gates: true},
		{name: "grounding alone", mode: "grounding", values: true, gates: true},
		{name: "deep", mode: "deep", values: true, gates: true},
		{name: "the fork's gate off", mode: "off", values: true},
		{name: "an sdd without the key", values: true},
		{name: "no sdd reader"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var values pieces.SddValues
			if tc.values {
				values = &test.SddValues{Values: map[string]string{"verify.mode": tc.mode}}
			}

			mode, gates := pieces.SddGates(context.Background(), values, t.TempDir())

			if gates != tc.gates {
				t.Errorf("sdd should count as gating graph writes only under a verify.mode other than off, got %v for %q", gates, mode)
			}

			if mode != tc.mode {
				t.Errorf("the mode sdd reports should come back as printed, got %q", mode)
			}
		})
	}
}

func TestTheSessionStartSaysSddHelpersIsDeprecated(t *testing.T) {
	cases := []struct {
		name    string
		mode    string
		options string
		graph   bool
		aside   bool
	}{
		{name: "the fork gating a graph project", mode: "prose", graph: true, aside: true},
		{name: "the fork's gate off", mode: "off", graph: true},
		{name: "an sdd without the key", graph: true},
		{name: "no graph", mode: "prose"},
		{name: "the gate off on the machine", mode: "prose", graph: true, options: `"grounding_gate_enabled":false`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := pieces.Env{Settings: settings(t, tc.options), Dir: t.TempDir()}

			if tc.graph {
				err := os.Mkdir(filepath.Join(env.Dir, ".sdd"), 0o755)
				if err != nil {
					t.Fatalf("creating the graph: %v", err)
				}
			}

			values := map[string]string{"sessions.token_saving": "on"}
			if tc.mode != "" {
				values["verify.mode"] = tc.mode
			}

			env.SddValues = &test.SddValues{Values: values}

			out := check(t, env)

			if strings.Contains(out, pieces.AsideLine(tc.mode)) != tc.aside {
				t.Errorf("the session start should say sdd-helpers stands aside only where sdd gates a graph project the gate works on, got:\n%s", out)
			}

			if strings.Contains(out, pieces.DeprecatedLine) == tc.aside {
				t.Errorf("the session start should say sdd-helpers is deprecated wherever it does not stand aside, got:\n%s", out)
			}

			if tc.aside && strings.Contains(out, pieces.TokenSavingLine) {
				t.Errorf("a gate standing aside collapses nothing, so the session start should not name sdd's own repeat filter, got:\n%s", out)
			}
		})
	}
}

func TestTheDeprecationLinesNameTheWayOut(t *testing.T) {
	lines := map[string]string{
		"deprecated":     pieces.DeprecatedLine,
		"standing aside": pieces.AsideLine("prose"),
	}

	for name, line := range lines {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(line, "`claude plugin uninstall sdd-helpers@sdd-helpers`") {
				t.Errorf("the line should name how to remove the plugin, got %q", line)
			}
		})
	}

	if !strings.Contains(pieces.DeprecatedLine, pieces.ForkRelease) ||
		!strings.Contains(pieces.DeprecatedLine, "`brew install --cask fza/tap/sdd`") {
		t.Errorf("the deprecation should name the fork release and how to install it, got %q", pieces.DeprecatedLine)
	}

	if !strings.Contains(pieces.CaptureAside("deep"), "verify.mode `deep`") {
		t.Errorf("sdd-capture standing aside should name the mode it stands aside for, got %q", pieces.CaptureAside("deep"))
	}
}
