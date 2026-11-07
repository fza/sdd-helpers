# sdd-helpers

Your project keeps its decisions in an [sdd](https://github.com/networkteam/sdd) graph, but your
coding agent does not read it unless made to. Asked about a past decision, it answers from the one
entry it happened to open, or from what it believes the graph holds. It drafts a new decision that
an older one already settled, reads an entry without the later one that renamed everything in it,
and captures entries by hand that break the project's own rules for the graph.

sdd-helpers fixes that, quietly, for every Claude Code session on your machine. Install it once: in
every project carrying a graph, the agent searches and lists the graph before it asks you a question
or drafts an entry, reads each entry with what refined it, and writes new entries through one checked
path. There is nothing to run by hand. Hooks do the work in the background, and a short skill per
piece tells the agent what it is doing. A project without a graph never hears from it.

sdd-helpers is a temporary workaround. Each piece works around a known quirk of stock sdd, and the
goal is to fix that quirk in sdd itself. Once stock sdd does a piece's job, the piece is removed
from sdd-helpers.

> **Your data stays on your machine.** sdd-helpers keeps its state in local files and sends none of
> it anywhere. The one request it makes is its launcher downloading the plugin's own release from
> GitHub; there is no update check. A capture runs sdd's own pre-flight, which reaches the model
> provider sdd is configured for, as any sdd write does.
> [plugin/README.md](plugin/README.md#what-it-sends) lists every case.

| Piece | Command | What it does |
|---|---|---|
| grounding gate | `sdd-grounding-gate` | Makes the agent read the decision graph before it asks, drafts or captures, keeps credentials out of the graph, and collapses sdd output the agent was already shown |
| capture | `sdd-capture` | Writes new graph entries from a draft file through one checked path |

## Features

### Grounding gate

- **Decisions are read before they are discussed.** The agent must search the graph both by meaning
  and by literal term, and list it, before it asks you a question, drafts an entry or captures one.
- **Entries are read with what changed them.** A read of an entry that leaves out what refined it is
  refused and echoed back at the depth owed.
- **No credential reaches the graph.** Every graph write, by command, draft, attached file or sdd's
  own MCP tools, is scanned first, since an entry is shared and never edited.
- **The project's rules hold on sdd's engine too.** A capture through sdd's MCP tools keeps the
  project's topic rule and draft lint.
- **Repeated sdd output shrinks.** An entry the agent was already shown comes back as a short pointer
  instead of in full.

### Capture

- **One checked path into the graph.** Every new entry passes the project's lint, sdd's pre-flight
  and a credential scan before it is written, the body handed to sdd whole, never through a shell.
- **A provider fault is not the draft's fault.** A failing pre-flight model call is retried, and can
  fall back to other models.
- **Drafts clean up after themselves.** A captured draft is deleted, unless git tracks it or it lies
  outside the work tree.

## Requirements

- macOS or Linux, on amd64 or arm64.
- Claude Code. The one-step `/plugin install` with `--marketplace` needs 2.1.275 or later.
- The [sdd](https://github.com/networkteam/sdd) CLI on the `PATH`, and a project carrying a graph
  under `.sdd/`. Collapsing repeated sdd output needs the fza fork, `v0.18.0-alpha.2.fza.41` or later
  (`sessions.token_saving`); with any other sdd, answers pass through as sdd gives them. Collapsing a
  `view` answer needs `v0.18.0-alpha.2.fza.42` or later (`view_sections`).

## Installing

```bash
claude plugin marketplace add fza/sdd-helpers
claude plugin install sdd-helpers@sdd-helpers
```

- **From inside a session**, one step does both:
  `/plugin install sdd-helpers --marketplace fza/sdd-helpers`.
- **To pin a release**, add the marketplace as `fza/sdd-helpers#v<version>`.
- **Installed at user scope**, the plugin runs in every project on the machine, and stays silent in
  each one without a graph.
- **Turn auto-update on**, so every release arrives by itself. Claude Code leaves it off for this
  marketplace; add this to `~/.claude/settings.json` (or turn it on under `/plugin`, Marketplaces):

  ```json
  {
    "extraKnownMarketplaces": {
      "sdd-helpers": {
        "source": {
          "source": "github",
          "repo": "fza/sdd-helpers"
        },
        "autoUpdate": true
      }
    }
  }
  ```

- **To update by hand**, run `claude plugin update sdd-helpers@sdd-helpers`. The update takes effect
  in the next session, or after `/reload-plugins`.

### sdd, for projects with a graph

```bash
brew install networkteam/tap/sdd
```

Without Homebrew:

```bash
curl -sL https://github.com/networkteam/sdd/releases/latest/download/install.sh | sh
```

**Optional: the fork [github.com/fza/sdd](https://github.com/fza/sdd)** (`brew install fza/tap/sdd`).
sdd-helpers works with stock sdd, and uses the fork's extras wherever the installed `sdd` offers
them. [`docs/plugin/usage.md`](docs/plugin/usage.md#sdd) lists them.

## Options

Options are set once per machine and apply to every project: in Claude Code's `/plugin` dialog
(sdd-helpers, Configure options), or `pluginConfigs` in `~/.claude/settings.json`. Managed settings
win over both. A project's own `.claude/settings*.json` is never read for these options.

```json
{
  "pluginConfigs": {
    "sdd-helpers@sdd-helpers": {
      "options": { "capture_remove_draft": false }
    }
  }
}
```

| Option | Default | What it controls |
|---|---|---|
| `grounding_gate_enabled` | `true` | The grounding gate. Off, it lets every call through, and `sdd-capture` goes off with it |
| `capture_enabled` | `true` | `sdd-capture`. Off, it refuses every capture |
| `capture_remove_draft` | `true` | `sdd-capture` deletes a draft once its entry is captured, unless git tracks it or it lies outside the work tree, the graph repository's work tree and `/tmp` |

A project lifts the gate for itself with `enabled: false` in `.sdd/grounding.yaml`, which also
carries the project's topic prefix, draft lint, read depths and read commands
([docs/grounding-gate/usage.md](docs/grounding-gate/usage.md#configuration-sddgroundingyaml)).
The layers each option is read from are in [docs/plugin/usage.md](docs/plugin/usage.md#options).

## Dependencies

What each part needs on the machine:

| Part | Needs |
|---|---|
| the grounding gate and `sdd-capture` | `sdd` on the `PATH`, in a project carrying `.sdd/` |
| `sdd-capture` | `git`, to tell whether a captured draft is tracked before deleting it |
| the launcher fetching a release | `curl`, `sha256sum` or `shasum`, `tar` |
| building from source | Go 1.27 or later; a release needs no Go |

Turning one piece off never breaks the other silently. A feature that relies on the piece switches
off with it, and each session start names it:

```text
sdd-helpers: grounding gate is off, so sdd-capture is off; turn grounding_gate_enabled on to restore it
```

| Turned off | Also off | Because |
|---|---|---|
| `grounding_gate_enabled` | `sdd-capture` | Its grounding check is the gate's hook; without it, a draft lands unchecked |

## Moving over from agentic-helpers

Both programs came from the agentic-helpers plugin, where they were named `agentic-grounding-gate`
and `agentic-capture`. Install sdd-helpers and switch the two pieces off in agentic-helpers, or
remove that plugin, so no project is gated twice.

- **The options keep their keys**: `grounding_gate_enabled`, `capture_enabled`,
  `capture_remove_draft`, read from `pluginConfigs["sdd-helpers@sdd-helpers"]`.
- **`.sdd/grounding.yaml` keeps its keys**, but for `exempt_seats`, which is gone with no
  replacement: `enabled: false` is the only per-project opt-out.
- **Scripts and instructions** naming `agentic-capture` name `sdd-capture` instead.

## Development

The rules live in [`AGENTS.md`](AGENTS.md), plus [`source/AGENTS.md`](source/AGENTS.md) for Go
code. How each piece is used and how it works inside: [`docs/index.md`](docs/index.md).

```bash
GOBIN="$PWD/.claude/bin" go install -C source/hooks ./cmd/...   # build every binary
cd source/hooks && go vet ./... && go test -race ./...           # run every Go test
scripts/linux-test                                               # the same suites on Linux, in a container
claude plugin validate . && claude plugin validate plugin        # the marketplace and plugin manifests
```

Never install the binaries into `~/go/bin` or another directory on your `PATH`: every project's
shell would run them ahead of the plugin. To run your checkout's binaries in your own sessions,
point `SDD_HELPERS_BIN_DIR` at the checkout's `.claude/bin`;
[`docs/plugin/usage.md`](docs/plugin/usage.md#working-on-a-checkout) explains how. Only the `release`
workflow cuts releases.

## License

MIT, Felix Zandanel. See [`LICENSE`](LICENSE).
