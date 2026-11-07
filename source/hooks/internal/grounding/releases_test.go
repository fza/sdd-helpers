package grounding_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/grounding"
	"github.com/fza/sdd-helpers/source/hooks/internal/hookio"
	"github.com/fza/sdd-helpers/source/hooks/internal/shellwalk"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

// Every committed release fixture holds the gate's state as a session of that
// release left it, and the gate calls that answer to it. The current gate
// answers each the same, so no release loses a session's reads, grounded
// drafts or shown sets.
func TestEveryCommittedReleaseAnswersAsItDid(t *testing.T) {
	for _, version := range test.ReleaseVersions(t) {
		t.Run(version, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}

			test.ReleaseTree(t, version, root)

			repo := filepath.Join(root, "repo")
			env := grounding.Env{
				ProjectDir: repo, WorkingDir: repo, Home: filepath.Join(root, "home"), Runner: &test.Runner{},
				Files:     shellwalk.OSFiles{},
				SddValues: &test.SddValues{Values: map[string]string{"sessions.token_saving": "off"}},
			}

			for _, call := range test.ReleaseDumpOf(t, version, root).Calls {
				var stdout, stderr bytes.Buffer

				code := grounding.Main(context.Background(), call.Args, env,
					hookio.Streams{In: strings.NewReader(call.Stdin), Out: &stdout, Err: &stderr})

				for _, successor := range successors {
					call.Stdout = successor(t, call)
				}

				if code != call.Code || stdout.String() != call.Stdout || stderr.String() != call.Stderr {
					t.Errorf("%v on %s should answer as %s did, got exit %d\n%s%s\nwant exit %d\n%s%s",
						call.Args, call.Stdin, version, code, stdout.String(), stderr.String(), call.Code, call.Stdout, call.Stderr)
				}
			}
		})
	}
}

// successors turn what a release printed for a call into what the current
// release prints for the same state, where a decision changed the answer's
// form on purpose. A call no successor changes keeps its answer as it was.
var successors = []func(t *testing.T, call test.ReleaseCall) string{
	plainCollapse,
}

// plainCollapse is the collapse answer of 20261107-185320-d-tac-1o0: the
// shortened text a release handed over in an MCP content wrapper under
// `updatedMCPToolOutput` arrives as a plain string under `updatedToolOutput`.
func plainCollapse(t *testing.T, call test.ReleaseCall) string {
	t.Helper()

	if !slices.Contains(call.Args, "collapse") || call.Stdout == "" {
		return call.Stdout
	}

	var wrapped struct {
		HookSpecificOutput struct {
			HookEventName string `json:"hookEventName"`
			Updated       *struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"updatedMCPToolOutput"`
		} `json:"hookSpecificOutput"`
	}

	err := json.Unmarshal([]byte(call.Stdout), &wrapped)
	if err != nil {
		t.Fatalf("a recorded collapse answer should be JSON: %v", err)
	}

	if wrapped.HookSpecificOutput.Updated == nil {
		return call.Stdout
	}

	if len(wrapped.HookSpecificOutput.Updated.Content) != 1 {
		t.Fatalf("a recorded collapse answer should hold one text block, got %s", call.Stdout)
	}

	var plain struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			UpdatedToolOutput string `json:"updatedToolOutput"`
		} `json:"hookSpecificOutput"`
	}

	plain.HookSpecificOutput.HookEventName = wrapped.HookSpecificOutput.HookEventName
	plain.HookSpecificOutput.UpdatedToolOutput = wrapped.HookSpecificOutput.Updated.Content[0].Text

	encoded, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("encoding the successor answer: %v", err)
	}

	return string(encoded)
}
