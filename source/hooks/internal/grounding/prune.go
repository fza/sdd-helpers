package grounding

import (
	"context"
	"fmt"
	"time"

	"github.com/fza/sdd-helpers/source/hooks/internal/hookio"
	"github.com/fza/sdd-helpers/source/hooks/internal/releaseprune"
)

const pruneSelf = "sdd-helpers: "

// startPrune tells a failed release prune once, then starts the day's run.
// It runs in the SessionStart hook, so the run is detached: no hook waits
// for it.
func startPrune(ctx context.Context, env Env, streams hookio.Streams) {
	dir := releaseprune.DataDir(env.Getenv)

	warning := releaseprune.Warning(dir)
	if warning != "" {
		_, _ = fmt.Fprintln(streams.Out, pruneSelf+warning)
	}

	_ = releaseprune.Start(ctx, dir, env.Starter, env.Executable, time.Now())
}

// prune runs `sdd-grounding-gate prune`: one release prune, bounded by
// releaseprune.RunLimit, printing each folder it deleted, and exiting 1 on
// an error the run recorded. Run by hand, outside the plugin's hooks, it
// names no data directory and touches nothing.
func prune(ctx context.Context, env Env, streams hookio.Streams) int {
	ctx, cancel := context.WithTimeout(ctx, releaseprune.RunLimit)
	defer cancel()

	result := releaseprune.Run(ctx, releaseprune.PluginReleases(env.Getenv, env.Processes), time.Now())

	for _, deleted := range result.Deleted {
		_, _ = fmt.Fprintln(streams.Out, "deleted "+deleted)
	}

	if result.Error != "" {
		_, _ = fmt.Fprintln(streams.Err, pruneSelf+result.Error)

		return 1
	}

	return 0
}
