package grounding_test

import (
	"bytes"
	"context"
	"path/filepath"
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

				if code != call.Code || stdout.String() != call.Stdout || stderr.String() != call.Stderr {
					t.Errorf("%v on %s should answer as %s did, got exit %d\n%s%s\nwant exit %d\n%s%s",
						call.Args, call.Stdin, version, code, stdout.String(), stderr.String(), call.Code, call.Stdout, call.Stderr)
				}
			}
		})
	}
}
