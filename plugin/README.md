# sdd-helpers

A Claude Code plugin for projects keeping an [sdd](https://github.com/networkteam/sdd) decision
graph: a gate that makes the agent read the graph before it asks, drafts or writes to it, and a
checked path for new graph entries. Each piece comes with a skill that explains it to the agent. The
full guide lives in the repository: <https://github.com/fza/sdd-helpers>.

## Install

```
/plugin install sdd-helpers --marketplace fza/sdd-helpers
```

Claude Code 2.1.275 or later takes this in one step; older releases run
`claude plugin marketplace add fza/sdd-helpers`, then
`claude plugin install sdd-helpers@sdd-helpers`. Claude Code keeps a third-party plugin on the
version first installed: turn auto-update on under `/plugin`, Marketplaces, or update with
`claude plugin update sdd-helpers@sdd-helpers`.

## Pieces

- **sdd-grounding-gate** holds questions, drafts and graph writes until the agent has read the
  project's graph, refuses a graph write holding what looks like a credential, and shortens sdd
  output the agent was already shown.
- **sdd-capture** writes new graph entries from a draft file through one checked path.

## Example prompts

- "What did we decide about the release workflow?": the gate has the agent search the graph first.
- "Capture that as a decision": the agent drafts the entry and captures it through sdd's engine, or
  through `sdd-capture` where the project serves no engine.

## What it runs

The hooks and the `sdd-capture` command run `hooks/run`, a shell script that starts the Go binaries
of the release named in `.claude-plugin/plugin.json`. On the first run of a release it downloads
them from `https://github.com/fza/sdd-helpers/releases`, checks each archive against the release's
`checksums.txt`, and keeps them under Claude Code's plugin data folder. The binaries,
`sdd-grounding-gate` and `sdd-capture`, are built from the repository's `source/hooks` by its release
workflow. The gate runs only as a hook, at session start and end and around tool calls, and once a
day a session start runs it in the background to delete old releases; `sdd-capture` runs only when
the agent calls it.

## What it sends

Nothing leaves the machine unless one of these applies:

- **A release is first run.** The binaries are fetched from GitHub, as above. The plugin asks
  GitHub nothing else, no update check included.
- **A graph entry is captured through `sdd-capture`.** sdd's pre-flight check sends the draft to the
  model provider sdd is configured for (`claude -p` unless set otherwise), as sdd itself does.

## What it writes outside the project

- The release binaries, one folder per version in the plugin's data directory, each with a `used`
  mark each session start running it writes. Once a day a release no session may still run goes:
  one the plugin does not name, no running `sdd-` program executes from and no session start used
  within 14 days. The run records its outcome in `release-prune.json` there, beside its lock files.
- `sdd-caps.json` under `~/.cache/sdd-helpers` (or `$XDG_CACHE_HOME/sdd-helpers`): what the installed
  `sdd` supports, read from its help text.

Inside a project carrying `.sdd/`, the gate keeps each session's reads, grounded drafts and shown
entries under `.sdd/tmp/grounding-gate/`, removed when the session ends; `sdd-capture` takes its lock
at `.sdd/tmp/capture.lock`, and deletes a captured draft unless `capture_remove_draft` is off.

## What it decides

- **The grounding gate** refuses graph writes, an sdd show short of the depth owed and an sdd call
  discarding its output until the agent has read the graph, and lets a question or a draft edit
  through with a note naming the reads still owed. It refuses a graph write holding what looks like
  a credential, whatever else holds. It shortens an sdd MCP answer to what the calling agent was not
  shown already, where sdd answers with `sessions.token_saving` off. A project lifts it with
  `enabled: false` in `.sdd/grounding.yaml`.
- **sdd-capture** refuses a draft whose header, credential scan, project lint or sdd pre-flight
  fails, and writes nothing then.

Each piece can be switched off in the plugin's options: `grounding_gate_enabled` (`sdd-capture`
goes off with it), `capture_enabled`, and `capture_remove_draft`.

## Support

Issues and questions: <https://github.com/fza/sdd-helpers/issues>. Licensed under MIT.
