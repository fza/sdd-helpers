package pieces_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/pieces"
	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func settings(t *testing.T, options string) []string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "settings.json")
	test.WriteFile(t, path, `{"pluginConfigs":{"sdd-helpers@sdd-helpers":{"options":{`+options+`}}}}`)

	return []string{path}
}

func check(t *testing.T, env pieces.Env) string {
	t.Helper()

	var captured test.Captured

	code := pieces.Check(context.Background(), env, captured.Streams(""))
	if code != 0 {
		t.Fatalf("check should exit 0, got %d: %s", code, captured.Err.String())
	}

	return captured.Out.String()
}

func TestEachTieBreaksOnlyWhileItsPieceIsOff(t *testing.T) {
	cases := []struct {
		off  string
		ties []pieces.Tie
	}{
		{`"grounding_gate_enabled":false`, []pieces.Tie{pieces.CaptureGrounded}},
		{`"grounding_gate_enabled":false,"capture_enabled":false`, nil},
		{`"capture_enabled":false`, nil},
	}

	for _, tc := range cases {
		t.Run(tc.off, func(t *testing.T) {
			options, err := pluginconfig.ReadWith(settings(t, tc.off), nil)
			if err != nil {
				t.Fatal(err)
			}

			broken := pieces.Broken(options)
			if len(broken) != len(tc.ties) {
				t.Fatalf("exactly the ties resting on the off piece, of pieces still on, should break, got %d", len(broken))
			}

			for index, tie := range tc.ties {
				if broken[index].Feature != tie.Feature || tie.Holds(options) {
					t.Errorf("the tie %q should be broken", tie.Feature)
				}
			}
		})
	}

	if broken := pieces.Broken(pluginconfig.Defaults()); len(broken) != 0 {
		t.Errorf("with every piece on, no tie should break, got %d", len(broken))
	}
}

func TestTheLinesAPieceAndATieSpeak(t *testing.T) {
	if got := pieces.CaptureGrounded.Line(); got != "sdd-helpers: grounding gate is off, so sdd-capture is off; turn grounding_gate_enabled on to restore it" {
		t.Errorf("a broken tie should name the piece off, the feature and the option, got %q", got)
	}

	if got := pieces.Capture.Off(); got != "sdd-capture: off on this machine (plugin option capture_enabled)" {
		t.Errorf("an off piece's command should name its option, got %q", got)
	}

	if got := pieces.CaptureGrounded.Off(); got != "sdd-capture: off while grounding gate is off (plugin option grounding_gate_enabled)" {
		t.Errorf("a command whose work rests on an off piece should say so, got %q", got)
	}
}

func TestCheckNamesEachBrokenTieAndUnreadableOptions(t *testing.T) {
	cases := []struct {
		name     string
		settings []string
		want     []string
	}{
		{name: "every tie whole says only the deprecation", settings: settings(t, "")},
		{name: "a broken tie", settings: settings(t, `"grounding_gate_enabled":false`), want: []string{pieces.CaptureGrounded.Line()}},
		{name: "options that cannot be read", settings: settings(t, `"capture_enabled":"sometimes"`),
			want: []string{"sdd-helpers: reading the sdd-helpers@sdd-helpers options in "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := check(t, pieces.Env{Settings: tc.settings})

			deprecated := "sdd-helpers: " + pieces.DeprecatedLine + "\n"
			if !strings.HasPrefix(out, deprecated) {
				t.Errorf("check should open on the deprecation line, got:\n%s", out)
			}

			if len(tc.want) == 0 && out != deprecated {
				t.Errorf("check should print nothing but the deprecation where nothing is wrong, got:\n%s", out)
			}

			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("check should print %q, got:\n%s", want, out)
				}
			}
		})
	}
}
