# The plugin: usage

## How a project opts in

**Nothing runs against a project until it carries an sdd graph.** Nothing is created, nothing is
asked, and a project without `.sdd/` never hears from the gate.

| Piece | Runs when the project carries | Otherwise |
|---|---|---|
| Grounding gate | a `.sdd/` decision graph, at the project root or above it, without `enabled: false` in its `grounding.yaml` | exits 0, silently |
| Capture | a `.sdd/` graph governing the draft or the working directory | `sdd-capture` exits 1: `no .sdd graph governs this draft or this directory` |

So opting a project in is creating its graph (`sdd init`), and opting out while keeping the graph is
`enabled: false` in `.sdd/grounding.yaml` ([grounding gate](../grounding-gate/usage.md#configuration-sddgroundingyaml)).
A project needs no other configuration, no allowlist entry and no marker file.

## The skills

A hook fires and an agent that has never met it has to work out what just happened. Each piece has a
skill carrying what it cannot say in a one-line refusal.

| Skill | Carries |
|---|---|
| `sdd-helpers:grounding` | the sdd reads that open the gate, how deep to read an entry and why, which reference kind is the sharp one, and the gate's note; in `references/`, the `.sdd/grounding.yaml` keys the reads follow and what every other refusal means |
| `sdd-helpers:capture` | which capture path a session takes, what an entry carries, and the project's rules on an engine capture; in `references/sdd-capture.md`, the draft's shape, what runs before `sdd-capture` writes an entry, and what each of its refusals and exit codes means |

A skill's `SKILL.md` holds what bears on every load; what bears only in one situation (a refusal,
a project key, the `sdd-capture` path where sdd's engine serves the project) sits in a reference
file `SKILL.md` links, read only when that situation shows up.

Each skill is the whole of its rules. A global instruction file names the skill and says to load it;
it never restates what the skill holds, because two copies drift and a reader obeys whichever they
opened. Each says in its own description when it applies, so an agent in a project without a graph
has no reason to open either. A plugin update brings every skill's new text with it.

## Commands

Claude Code's Bash tool finds `sdd-capture` through the plugin's `bin/`, whose shim hands each call
to the launcher:

```sh
exec "$(dirname "$0")/../hooks/run" --command sdd-capture "$@"
```

So `sdd-capture --help` is answered by the real binary. A script outside Claude Code has no plugin on
its `PATH` and calls the binary by its full path: a release's, or one a checkout built into its own
`.claude/bin` ([working on a checkout](#working-on-a-checkout)).

An `sdd-capture` or `sdd-grounding-gate` binary on the `PATH` outside a plugin folder runs ahead of
the plugin in every project's Bash, an unreleased build included, so each session start names its
directory ([what each session start says](#what-each-session-start-says)).

## Where the binaries come from

The plugin holds no binaries. Every hook and command runs through `plugin/hooks/run`, which runs the
binary of the release `plugin/.claude-plugin/plugin.json` names, kept under the plugin's data
directory (`~/.claude/plugins/data/sdd-helpers-sdd-helpers/<version>/`). A session start, or a
command an agent runs, downloads a missing release from GitHub and checks it against the release's
`checksums.txt`; a prompt or a tool-call hook never downloads. How it fetches, locks and falls back
is in the [architecture](architecture.md#where-the-binaries-come-from).

Each session start writes a `used` mark into the folder of the release it runs. Once a day, a
session start in any project starts `sdd-grounding-gate prune` in the background, which deletes
every release folder no session may still need: it keeps the release the plugin names, every release
a running `sdd-` program executes from, and every release a session start used within 14 days. The
run records its outcome in `release-prune.json` in the data directory; what it keeps, and how, is in
the [architecture](architecture.md#release-pruning).

With no binary installed and none fetchable, the grounding gate refuses every call it guards rather
than let work run ungrounded, `sdd-capture` exits 1, and every other hook does nothing:

    sdd-helpers: no sdd-grounding-gate binary is installed, so the grounding gate refuses every call; start a session with network so release v<version> can be fetched, or set SDD_HELPERS_BIN_DIR
    sdd-helpers: no sdd-capture binary is installed; start a session with network so release v<version> can be fetched, or set SDD_HELPERS_BIN_DIR

A plugin naming no release yet, before the first one is cut, fetches nothing, and both lines end
`this plugin names no release yet; set SDD_HELPERS_BIN_DIR to a directory holding a build` instead.
The session start that failed to fetch says why, once.

## What each session start says

`sdd-grounding-gate check` runs at every session start, whether or not the gate is on, and prints
nothing where all is well. Each line opens `sdd-helpers: `:

| Line | When |
|---|---|
| `sdd-helpers: grounding gate is off, so sdd-capture is off; turn grounding_gate_enabled on to restore it` | `grounding_gate_enabled` is off while `capture_enabled` is on |
| `sdd-helpers: <why>`, naming the settings file and the key | a settings file cannot be read or parsed, or sets an option to no boolean; every option then reads as its default |
| ``sdd-helpers: sdd's sessions.token_saving is on, so the grounding gate's per-agent collapse of repeated sdd output is off; `sdd config set sessions.token_saving off` turns it on`` | the project carries a graph, the gate is on, and `sdd config get sessions.token_saving` prints `on` ([collapsing](../grounding-gate/usage.md#repeated-sdd-output-collapses)) |
| `sdd-helpers: <dir> on the PATH holds <names>, which every project's Bash runs ahead of the plugin; remove them` | a `PATH` directory outside `/.claude/plugins/` holds `sdd-capture` or `sdd-grounding-gate`; one line per directory |

The session start adds one line, once, after a release prune failed, in any project:

| Line | When |
|---|---|
| `sdd-helpers: release prune failed at <time>: <error>; see <data>/release-prune.json` | the last run recorded an error no session start has told yet |

## sdd

The grounding gate and `sdd-capture` work with stock sdd
([github.com/networkteam/sdd](https://github.com/networkteam/sdd)) and use what the fork
[github.com/fza/sdd](https://github.com/fza/sdd) adds wherever the installed `sdd` carries it. What a
binary carries is read from its own `sdd --help`, `sdd new --help`, `sdd init --help` and
`sdd info --help`, never from its version, and kept in `sdd-caps.json` under
`$XDG_CACHE_HOME/sdd-helpers` (`~/.cache/sdd-helpers` where unset), per binary path, modification
time and size.

| Fork surface | Without it |
|---|---|
| `sdd new --preflight-verified` | `sdd-capture` runs one `sdd new`, which checks and writes |
| `sdd new --title`, `--actor-kind` | a draft carrying `title` or `actor_kind` is refused, naming the fork |
| `sdd --local-config` | `--preflight-route` is refused |
| `sdd migrate` | a graph on schema 2 is refused; a schema refusal names no `sdd migrate` |
| `sdd init --project-dir` | the gate's engine refusal skips a project naming `project_dir` |
| `sdd info --format json` | `sdd-capture` reads a titled draft's graph schema from `.sdd/meta.json` instead of asking sdd |

## Options

Every option is set once per machine; no project overrides one. Each is a switch, on unless set:

| Option | Switches |
|---|---|
| `grounding_gate_enabled` | the grounding gate; off, `sdd-capture` goes off with it |
| `capture_enabled` | `sdd-capture` |
| `capture_remove_draft` | `sdd-capture` deleting a draft once its entry is captured ([capture](../capture/usage.md#the-draft-after-a-capture)) |

Each is read in layers, lowest first, the last one setting a key winning:

1. the built-in default;
2. the user's settings, `settings.json` under `CLAUDE_CONFIG_DIR` (else `~/.claude`), at
   `pluginConfigs["sdd-helpers@sdd-helpers"].options`, where the plugin's configuration dialog
   (`/plugin`) saves its answers;
3. in a hook, the options Claude Code hands it as `CLAUDE_PLUGIN_OPTION_<KEY>`;
4. Claude Code's managed settings, `managed-settings.json` and the `managed-settings.d/*.json`
   drop-ins in name order (darwin `/Library/Application Support/ClaudeCode`, linux
   `/etc/claude-code`), last, so no environment overrides what a machine locks.

A project's own `.claude/settings*.json` is never read: a project's committed files must not switch
a guard off for the machine. An empty value counts as unset at every layer. A value is a JSON
boolean or the string `true` or `false`. A command the Bash tool runs gets no
`CLAUDE_PLUGIN_OPTION_<KEY>`, so `sdd-capture` reads the settings files themselves. A settings file
that cannot be read or parsed, or a value that is no boolean, leaves every option at its default,
and the next session start names it.

```json
{
  "pluginConfigs": {
    "sdd-helpers@sdd-helpers": {
      "options": { "capture_remove_draft": false }
    }
  }
}
```

## Switching a piece off

Off, the grounding gate lets every call through and keeps no evidence, and `sdd-capture` exits 1:

    sdd-capture: off on this machine (plugin option capture_enabled)
    sdd-capture: off while grounding gate is off (plugin option grounding_gate_enabled)

The second holds while `grounding_gate_enabled` is off, since `sdd-capture`'s grounding check is the
gate's hook and a draft would land ungrounded; each session start names it. A skill stays listed
while its piece is off.

## Updates

Claude Code leaves plugin auto-update off for every third-party marketplace, and a marketplace's
`marketplace.json` has no field to change that. A machine takes a new release by
`claude plugin update sdd-helpers@sdd-helpers`, effective from the next session or after
`/reload-plugins`, or turns auto-update on for the marketplace in `~/.claude/settings.json` (or under
`/plugin`, Marketplaces):

```json
{"extraKnownMarketplaces": {"sdd-helpers": {"source": {"source": "github", "repo": "fza/sdd-helpers"}, "autoUpdate": true}}}
```

The plugin never asks GitHub for newer releases itself, and never writes `autoUpdate` into any
settings file.

## Working on a checkout

Add the checkout as the marketplace, so the plugin loads from it in place and an edit to a skill or
to `plugin/` reaches the next session or `/reload-plugins`:

```bash
claude plugin marketplace add /path/to/sdd-helpers
claude plugin install sdd-helpers@sdd-helpers
```

The binaries the checkout builds reach the hooks through `SDD_HELPERS_BIN_DIR`, set to the
checkout's gitignored `.claude/bin` in the `env` block of the checkout's own
`.claude/settings.local.json`:

```json
{ "env": { "SDD_HELPERS_BIN_DIR": "/path/to/sdd-helpers/.claude/bin" } }
```

A change to a binary then reaches the hooks on the next event after
`GOBIN="$PWD/.claude/bin" go install -C source/hooks ./cmd/...` from the checkout root. Where the
variable is set, no release is fetched or run. Never install into `~/go/bin` or another directory on
the `PATH`: every project's Bash calls would find the build there ahead of the plugin. Set in
`~/.claude/settings.json` instead, the variable would hold every project on the machine to the last
`go install` build: a release would reach no session until installed, and an install from a tree
holding unfinished work would run it everywhere.

## Moving from agentic-helpers

Both programs came from the agentic-helpers plugin, where they were named `agentic-grounding-gate`
and `agentic-capture`. The options and every `.sdd/grounding.yaml` key the gate reads keep their
names. `enabled: false` is the only way a project opts out of the gate, and only a `prc`-layer rules
entry is read shallower than `show_depth` owes.
