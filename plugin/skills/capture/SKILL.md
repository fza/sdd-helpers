---
name: capture
description: 'Capturing an sdd graph entry: through sdd''s engine (mcp__sdd__start_procedure, canonical capture) wherever mcp__sdd__* tools are present, through sdd-capture from a draft file only where they are not. Draft shape, project lint, pre-flight, every refusal. Use before any capture, and whenever sdd-capture or the grounding gate refuses one. Only projects carrying a .sdd/ graph.'
license: MIT
---

# Capture

## Which way

| Session has | Capture through |
|---|---|
| `mcp__sdd__*` tools (sdd's engine) | `mcp__sdd__start_procedure`, `canonical: capture`, driven by the `sdd` skill. Never `sdd-capture` |
| no sdd MCP server, or one refusing the graph | `sdd-capture <draft.md>`, below |

- **Engine plays the entry back** for the owner's confirmation, runs a writing-guide review, checks
  the generated summary. `sdd-capture` does none.
- **Gate enforces it.** `.mcp.json` declares `mcpServers.sdd` → `sdd-capture` refused, whatever
  reads stand: `GROUNDING GATE: refused. This project serves sdd's engine ...`. Use the engine.
  `cli_capture: true` in `.sdd/grounding.yaml` lifts it: owner's setting, never yours. An sdd server
  at user or local scope, outside `.mcp.json`, goes unseen.
- **Either path owes the grounding reads** (`sdd-helpers:grounding`).
- **Never a credential in an entry.** Gate scans every draft, attached file, engine report, start
  params and staged file; its refusals: `sdd-helpers:grounding`. Name the variable or file instead,
  capture again.

## Engine capture: the project's rules

sdd's guide and pre-flight never apply `.sdd/grounding.yaml`. Gate does, on the capture's start and
every `mcp__sdd__next` to a capture it saw start (`params`, `report`, a chooser's `fields`).

| Rule | Refused as | Fix |
|---|---|---|
| `listing_prefix` | ``GROUNDING GATE: refused. These `topics` carry none opening with `<prefix>` ...``; playback `confirm` while none held: ``... This capture holds no topic opening with `<prefix>` ...`` | report `topics` with one `<prefix>` topic; at playback, `adjust` |
| `draft_lint` | ``... The project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) reports on the body this call leaves``, findings named `body:<line>:...` | fix each, report again |
| lint failing silent | ``... did not read the body this call leaves: <why>`` | make the lint run, call again |
| `bodyPatch`, no body held | ``... This `bodyPatch` edits a body the gate holds no record of`` | report `body` in full once |
| `bodyHandle`, no staging seen | see below | stage again, or report `body` |

- **`bodyPatch`** → applied to the last whole body by sdd's rule, result linted.
- **`bodyHandle`** → `draft_lint` set: gate lints the text it recorded at `mcp__sdd__stage_attachment`
  (`content`, the file `path` names, `patches` applied as sdd applies them), leading `---` header
  block left out. Handle staged before the gate saw it, or ledger lost →
  ``GROUNDING GATE: refused. This `bodyHandle` names `<handle>`, a staged file the gate holds no record of, so the project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) cannot read the body.``
  Asked once per handle.

## sdd-capture

`sdd new "$(cat body.md)"` → second shell re-reads the body → backticks run as commands, every code
identifier gone from an immutable entry. `sdd-capture` passes the body as one argument. Outside
the engine: always it, never `sdd new` by hand.

**Only a project carrying `.sdd/`.** None governing draft or working directory →
`no .sdd graph governs this draft or this directory`, exit 1.

```bash
sdd-capture <draft.md>                   # lint, pre-flight, capture
sdd-capture <draft.md> --force           # capture despite a high finding. Owner's call only
sdd-capture <draft.md> --skip-preflight  # entry marked unchecked. Only where the tool advises it
sdd-capture <draft.md> --preflight-limit 8m   # model call gets 8m per try (sdd --preflight-timeout). Default: max(5m, llm.timeout)
CAPTURE_PREFLIGHT_LIMIT=8m sdd-capture <draft.md>  # same, for a caller. The flag wins
sdd-capture <draft.md> --preflight-route glm-5.3=<layer.yaml>  # fallback for the pre-flight. Repeatable
sdd-capture <draft.md> --preflight-done <sha256>  # caller ran the pre-flight on these bytes already
sdd-capture --help                       # usage + one line per flag, stdout, exit 0
```

- **`--preflight-done <sha256>`** = sha256 of the draft file's bytes as read from disk, front matter
  included. Match → header and lint still run, pre-flight skipped,
  `pre-flight skipped: the caller ran it on these exact bytes`, entry recorded verified, never
  unchecked. Mismatch → `--preflight-done names <given>, but the draft hashes to <actual>; running
  the pre-flight`, then the pre-flight. Beside `--skip-preflight` → usage, exit 1.
- **No binary** → `sdd-helpers: no sdd-capture binary is installed; ...`, exit 1. User's to fix.

## Draft

```markdown
---
type: d
layer: tac
kind: directive
intent: pending
confidence: high
participants: [Felix]
topics: [area-hooks]
refs:
  - {id: 20260925-144721-d-tac-a9x, kind: refines, desc: why it points there}
---

Body. Markdown, unfenced.
```

- **Header = `sdd new`'s arguments.** `type`, `layer`, `confidence` required. `kind`, `intent`,
  `canonical`, `actor`, `actor_kind`, `class`, `summary`, `title` → one value. `topics`, `closes`,
  `supersedes`, `participants`, `aliases`, `actors` → list. `refs`, `involvement`, `topic` → one
  object per item. `when`, `index` → one object. `attach` → list of paths. Value checks → `sdd new`.
- **Any other key → refused, exit 1**, naming it, listing the accepted ones. Nothing runs. Never a
  silent drop: an entry written without a field stays without it.
- **Draft lives where the project keeps drafts.** `.tmp/drafts/` where `.tmp/` exists.
- **Draft and every attached file exist before the capture's call**, written in a call of their
  own. Attach by literal path: no `$` or `~`; relative only to the project directory, with no `cd`
  or launcher in the call. Never `-:<name>` (standard input). Capture as its own plain call.
- **Never fence the body.** A lint reads a fenced body as code, reports nothing.
- **Plain prose.** People read the graph.
- **At least one topic.** Project naming a `listing_prefix` → one topic carrying it.
- **Name every surface the decision creates**: command, flag, RPC verb, config key, label, storage
  path, environment variable. "We'll add a command" → not settled. Exact spelling → settled.
- **No acceptance criterion waits on a decision the entry could take itself.**
- **Body carries what, why, rejected alternatives with their reasoning, accepted costs.** No
  rejected alternatives → re-proposed within the month.
- **Ref kinds** → `sdd-helpers:grounding`. Immutable: right before capture.

## What runs, in order

| Step | Blocks when |
|---|---|
| grounding gate | project serves sdd's engine, no `cli_capture: true`; or reads owed, draft not grounded |
| header | no `type`/`layer`/`confidence`; unknown key; `listing_prefix` set, no topic carries it |
| credential scan | body, header value or attached file looks like a credential: `sdd-capture: refused. The capture holds what looks like a credential (<rule ids>, <where>, line <n>); the graph is shared and immutable`. Unreadable or `-:<name>` attachment → same prefix, the gate's wording (`sdd-helpers:grounding`). Runs again right before each `sdd new` |
| sdd | none → `sdd-capture needs sdd on the PATH`, exit 1. Draft or call needs a fork-only surface the installed sdd lacks → refused naming `brew install fza/tap/sdd`, exit 1, nothing runs. Drop the key, or the owner installs the fork |
| project lint | `draft_lint` set and it reports; or it fails, says nothing |
| pre-flight | `sdd new --dry-run` refuses; any `[high]` finding. Provider fault → retried, below. Skipped on a `--preflight-done` match |
| capture | `sdd new --preflight-verified`, under `.sdd/tmp/capture.lock`. Runs once, never retried |

- **Interrupted** → `sdd-capture: interrupted before sdd ran: <why>; nothing was written`.
- **Stock sdd (no `--preflight-verified`)** → no dry run: one `sdd new` under the lock checks and
  writes. `[high]` → exit 2, nothing written. Provider fault → retried. Cut off by the limit → exit
  1, never retried: `git log` first. `--force` past a `[high]` → written with `--skip-preflight`,
  recorded unchecked.
- **Fork-only:** draft keys `title`, `actor_kind`; `--preflight-route`; `--preflight-done`; a graph
  on schema 2 or later. Read from sdd's help text, never its version.
- **`title` on a graph without titles** → refused before anything runs: ``the graph is on schema 1,
  which has no entry titles; drop the draft's `title` key, or run `sdd migrate` to upgrade the
  graph``. Drop the key; `sdd migrate` is the owner's call. Asked of `sdd info --format json` where
  sdd has it, else read from `.sdd/meta.json`.
- **`the graph is on a schema newer than this sdd writes`** → sdd older than the graph: install the
  newer sdd. **`whether the graph takes entry titles is unknown:`** → `sdd info` failed, its output
  follows. Neither is the draft's fault.
- **Split layout** (`project_dir` in the code repo's `.sdd/config.yaml`) → `grounding.yaml` and the
  lock are the graph repo's. sdd and the lint run from the code repo.
- **`.sdd/grounding.yaml`:** `listing_prefix: area-` → topic rule. `draft_lint: <command>` → run from
  the project root on `.<draft>.body.md` beside the draft, findings shifted to the draft's lines. No
  key → no rule, no lint.
- **`[medium]` → shown, captures anyway.** Read each. Fold what holds, say why for what does not.
- **`[low]` → never shown.**
- **Exit 2 = pre-flight held it.** Fix the draft, run again. `--force` → owner decides, never you.
- **Exit 1 = anything else.** The message names it.

## Provider fault

Pre-flight = model call. Provider failing → not the draft's fault → retried, not refused.

| Fault | sdd output carries |
|---|---|
| answered nothing | `no content or tool calls in response` |
| rate-limited | `RateLimitError`, status 429 |
| failed with status | `API error: status code` 408, 409, 5xx |
| timed out | `timed out`, `no answer within llm.timeout` (the limit reaches sdd as `--preflight-timeout`) |
| no answer within the limit | try ran past limit + sdd's extraction bound + 30s (sdd lacking `--preflight-timeout`: the limit alone); sdd and its children killed |

- **Three tries: waits 10s, then 30s, then gives up.** One stderr line per retry:
  `the pre-flight's provider answered nothing; trying again in 10s`.
- **Gave up → exit 1:** `... on all 3 tries; the draft did not cause it, so capture it again later`,
  then sdd's output of the last try, if any. Never edit the draft for it. Run again later.
- **Gave up on timeouts** → also `a long draft may need more than <limit>; run again with a longer
  --preflight-limit`. Do that: the same limit fails the same way.
- **Engine capture fails `claude -p timed out` or `no answer within llm.timeout`** → no flag reaches
  it, only sdd's `llm.timeout`. Raising it (`sdd config set --local llm.timeout <duration>`) is the
  owner's call: tell them.
- **Fallback routes** (`--preflight-route <model>=<layer>`, in order given): primary gave up → each
  route one try, no wait. sdd gets `--model <model> --local-config <layer>`. First answering decides,
  its findings counting as the primary's. Capture step stays on the caller's own config.
  - Routes named → stdout opens `SDD_CAPTURE_ROUTE <model>` (or `primary`).
  - Every route failed → `route <name>: <fault>: <last line>` per route, then `... failed on every
    route; the draft did not cause it, so capture it again later`. Exit 1.
  - Layer missing, unreadable, not yours, or group/other-readable → `route <model> skipped: <layer>
    <why>`. Never read, never printed: holds an API key.
- **Never retried:** a `[high]` finding, `validation failed`, status 400/401/403/404, anything else.
  Refusal checked first: a `[high]` finding quoting "timed out" is still a refusal.
- **Graph on an older schema** → ``the graph needs `sdd migrate` before sdd writes to it; the draft
  did not cause this``, then sdd's reason, exit 1. Stock sdd, no `migrate` → ``sdd refuses to write
  this graph; the draft did not cause this``. Either step; never retried, never routed. Fix = `sdd
  migrate`: owner's call, rewrites the whole graph, asks y/N. Never edit the draft for it.
- **Capture step never retried.** Its one model call (summary) fails soft, before the write. Failure
  there = file or git; entry may be on disk. `git log` before running again.

## After

- **Read the generated summary.** Drift (wrong actor, wrong ref attribution, lost identifier) →
  `sdd summarize <id> --text '<faithful summary>'`. Never needs the draft.
- **Captured (exit 0) → draft deleted**, stderr `draft removed: <path>`. Gone for good: never
  recreate it, never look for it.
- **Kept, with a line saying why:**

  | `draft kept: <path> ...` | Means | Do |
  |---|---|---|
  | `is tracked by git` | removal = git change nobody asked for | remove in its own commit, if wanted |
  | `is outside the work tree` | outside working tree, graph repo's tree and `/tmp`: may be another's | leave, or remove once sure it is yours |
  | `is not a regular file` | symlink or no plain file: never deleted through | link and target untouched |
  | `changed while it was checked` | another file took the path mid-capture | look before removing anything |
- **Kept, no line:** plugin option `capture_remove_draft` off; git could not say whether it tracks
  the draft; or the deletion failed. Delete it once the entry reads right.
- **Failed (exit 1 or 2) → draft always stays**, no line: the copy to fix and run again.

## Off on the machine

`sdd-capture: off on this machine (plugin option capture_enabled)`, or
`sdd-capture: off while grounding gate is off (plugin option grounding_gate_enabled)`. Exit 1,
nothing runs. Tell the user; never capture with `sdd new` instead.
