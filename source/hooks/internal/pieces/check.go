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

// Check runs `sdd-grounding-gate check`, the SessionStart hook: the
// deprecation line in every project, one line per broken tie, one where the
// options cannot be read, one while sdd's own repeat filter is on, and one per
// directory on the PATH holding an sdd-helpers binary. It always exits 0: a
// SessionStart hook failing changes nothing the lines would.
func Check(ctx context.Context, env Env, streams hookio.Streams) int {
	options, err := pluginconfig.ReadWith(env.Settings, env.Getenv)
	if err != nil {
		options = pluginconfig.Defaults()
	}

	mode, aside := sddGatesHere(ctx, env, options)
	if aside {
		_, _ = fmt.Fprintln(streams.Out, self+": "+AsideLine(mode))
	} else {
		_, _ = fmt.Fprintln(streams.Out, self+": "+DeprecatedLine)
	}

	if err != nil {
		_, _ = fmt.Fprintln(streams.Out, self+": "+err.Error())
	}

	for _, tie := range Broken(options) {
		_, _ = fmt.Fprintln(streams.Out, tie.Line())
	}

	if !aside && tokenSavingOn(ctx, env, options) {
		_, _ = fmt.Fprintln(streams.Out, self+": "+TokenSavingLine)
	}

	for _, line := range pathBinaryLines(env.Getenv) {
		_, _ = fmt.Fprintln(streams.Out, self+": "+line)
	}

	return 0
}

// gated is whether the grounding gate works in the project: it is on and
// the project carries a graph.
func gated(env Env, options pluginconfig.Options) bool {
	return GroundingGate.On(options) && graphconfig.Resolve(env.Dir).Dir != ""
}

// sddGatesHere is SddGates in a project the grounding gate works on; where it
// does not, there is nothing to stand aside from.
func sddGatesHere(ctx context.Context, env Env, options pluginconfig.Options) (string, bool) {
	if !gated(env, options) {
		return "", false
	}

	return SddGates(ctx, env.SddValues, env.Dir)
}

// tokenSavingOn is whether, in a project carrying a graph the grounding gate
// works on, sdd reports its own repeat filter on. An sdd without the key
// prints nothing, and there the key cannot be set, so no line names it.
func tokenSavingOn(ctx context.Context, env Env, options pluginconfig.Options) bool {
	if env.SddValues == nil || !gated(env, options) {
		return false
	}

	return env.SddValues.SddValue(ctx, env.Dir, "sessions.token_saving") == "on"
}
