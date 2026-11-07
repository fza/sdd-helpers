package pieces

import (
	"context"
	"fmt"

	"github.com/fza/sdd-helpers/source/hooks/internal/graphconfig"
	"github.com/fza/sdd-helpers/source/hooks/internal/hookio"
	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
)

// Env says where the plugin's options are read, and the project the session
// works in, Dir, for the line saying sdd's own repeat filter keeps the
// grounding gate's collapse off.
type Env struct {
	Settings []string
	Getenv   func(string) string
	// SddValues reads one sdd setting in Dir; nil reads none.
	SddValues SddValues
	Dir       string
}

// SddValues answers what `sdd config get <key>` prints in dir, or nothing
// where sdd is missing, fails or knows no such key.
type SddValues interface {
	SddValue(ctx context.Context, dir string, key string) string
}

// TokenSavingLine is the SessionStart line while sdd's own repeat filter is
// on: sdd then answers without the structured fields the grounding gate
// collapses by.
const TokenSavingLine = "sdd's sessions.token_saving is on, so the grounding gate's per-agent collapse of " +
	"repeated sdd output is off; `sdd config set sessions.token_saving off` turns it on"

// Check runs `sdd-grounding-gate check`, the SessionStart hook: one line per
// broken tie, one where the options cannot be read, one while sdd's own
// repeat filter is on, and one per directory on the PATH holding an
// sdd-helpers binary. It prints nothing where nothing is wrong, and always
// exits 0: a SessionStart hook failing changes nothing the lines would.
func Check(ctx context.Context, env Env, streams hookio.Streams) int {
	options, err := pluginconfig.ReadWith(env.Settings, env.Getenv)
	if err != nil {
		_, _ = fmt.Fprintln(streams.Out, self+": "+err.Error())

		options = pluginconfig.Defaults()
	}

	for _, tie := range Broken(options) {
		_, _ = fmt.Fprintln(streams.Out, tie.Line())
	}

	if tokenSavingOn(ctx, env, options) {
		_, _ = fmt.Fprintln(streams.Out, self+": "+TokenSavingLine)
	}

	for _, line := range pathBinaryLines(env.Getenv) {
		_, _ = fmt.Fprintln(streams.Out, self+": "+line)
	}

	return 0
}

// tokenSavingOn is whether, in a project carrying a graph the grounding gate
// works on, sdd reports its own repeat filter on. An sdd without the key
// prints nothing, and there the key cannot be set, so no line names it.
func tokenSavingOn(ctx context.Context, env Env, options pluginconfig.Options) bool {
	if env.SddValues == nil || !GroundingGate.On(options) || graphconfig.Resolve(env.Dir).Dir == "" {
		return false
	}

	return env.SddValues.SddValue(ctx, env.Dir, "sessions.token_saving") == "on"
}
