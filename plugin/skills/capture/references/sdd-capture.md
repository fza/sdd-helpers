# sdd-capture

Only where no sdd MCP server serves the project. `sdd new "$(cat body.md)"` → a second shell
re-reads the body → backticks run as commands, every code identifier gone from an immutable entry.
`sdd-capture` passes the body as one argument: always it, never `sdd new` by hand. No `.sdd/`
governing draft or working directory → `no .sdd graph governs this draft or this directory`, exit 1.

```bash
sdd-capture <draft.md>                   # lint, pre-flight, capture
sdd-capture <draft.md> --force           # capture despite a high finding. Owner's call only
sdd-capture <draft.md> --skip-preflight  # entry marked unchecked. Only where the tool advises it
sdd-capture <draft.md> --preflight-limit 8m   # 8m per try (sdd --preflight-timeout). Default: max(5m, llm.timeout)
CAPTURE_PREFLIGHT_LIMIT=8m sdd-capture <draft.md>  # same, for a caller. The flag wins
sdd-capture <draft.md> --preflight-route glm-5.3=<layer.yaml>  # fallback for the pre-flight. Repeatable
sdd-capture <draft.md> --preflight-done <sha256>  # caller ran the pre-flight on these bytes already
sdd-capture --help                       # usage + one line per flag, stdout, exit 0
```

- **`--preflight-done <sha256>`** = sha256 of the draft file's bytes, front matter included. Match →
  header and lint still run, `pre-flight skipped: the caller ran it on these exact bytes`, entry
  recorded verified. Mismatch → `--preflight-done names <given>, but the draft hashes to <actual>;
  running the pre-flight`. Beside `--skip-preflight` → usage, exit 1.
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

- **Header = `sdd new`'s arguments.** `type`, `layer`, `confidence` required. One value: `kind`,
  `intent`, `canonical`, `actor`, `actor_kind`, `class`, `summary`, `title`. List: `topics`,
  `closes`, `supersedes`, `participants`, `aliases`, `actors`. One object per item: `refs`,
  `involvement`, `topic`. One object: `when`, `index`. `attach`: list of paths.
- **Any other key → refused, exit 1**, naming it and the accepted ones. Never a silent drop.
- Draft lives where the project keeps drafts: `.tmp/drafts/` where `.tmp/` exists.
- Draft and every attached file written in a call of their own before the capture. Attach by literal
  path: no `$` or `~`, relative only to the project directory, no `cd` or launcher in the call, never
  `-:<name>`. Capture as its own plain call.

## What runs, in order

| Step | Blocks when |
|---|---|
| grounding gate | project serves sdd's engine, no `cli_capture: true`; or reads owed, draft not grounded |
| header | no `type`/`layer`/`confidence`; unknown key; `listing_prefix` set, no topic carries it |
| credential scan | body, header value or attached file looks like a credential: `sdd-capture: refused. The capture holds what looks like a credential (<rule ids>, <where>, line <n>); the graph is shared and immutable`. Unreadable or `-:<name>` attachment → same prefix, the gate's wording (`sdd-helpers:grounding`). Runs again right before each `sdd new` |
| sdd | none → `sdd-capture needs sdd on the PATH`, exit 1. A fork-only surface the installed sdd lacks → refused naming `brew install fza/tap/sdd`, exit 1. Drop the key, or the owner installs the fork |
| project lint | `draft_lint` reports, or fails saying nothing. Run from the project root on `.<draft>.body.md` beside the draft, findings shifted to the draft's lines |
| pre-flight | `sdd new --dry-run` refuses; any `[high]` finding. Provider fault → retried. Skipped on a `--preflight-done` match |
| capture | `sdd new --preflight-verified`, under `.sdd/tmp/capture.lock`. Runs once, never retried |

- **Exit 2** = pre-flight held it: fix the draft, run again. `--force` → owner decides, never you.
  **Exit 1** = anything else; the message names it.
- `[medium]` → shown, captured anyway: fold what holds, say why for what does not. `[low]` → never
  shown.
- `sdd-capture: interrupted before sdd ran: <why>; nothing was written`.
- **Stock sdd** (no `--preflight-verified`) → one `sdd new` under the lock checks and writes. `[high]`
  → exit 2. Cut off by the limit → exit 1, never retried: `git log` first. `--force` past a `[high]`
  → written with `--skip-preflight`, recorded unchecked.
- **Fork-only:** draft keys `title`, `actor_kind`; `--preflight-route`; `--preflight-done`; a graph
  on schema 2 or later. Read from sdd's help text, never its version.
- **`title` on schema 1** → ``the graph is on schema 1, which has no entry titles; drop the draft's
  `title` key, or run `sdd migrate` to upgrade the graph``. Drop the key; `sdd migrate` is the
  owner's call.
- `the graph is on a schema newer than this sdd writes` → install the newer sdd.
  `whether the graph takes entry titles is unknown:` → `sdd info` failed, its output follows.
  Neither is the draft's fault.
- **Split layout** (`project_dir` in the code repo's `.sdd/config.yaml`) → `grounding.yaml` and the
  lock are the graph repo's; sdd and the lint run from the code repo.

## Provider fault

Not the draft's fault → retried, never refused. Three tries: wait 10s, then 30s. One stderr line
per retry: `the pre-flight's provider answered nothing; trying again in 10s`.

| Fault | sdd output carries |
|---|---|
| answered nothing | `no content or tool calls in response` |
| rate-limited | `RateLimitError`, status 429 |
| failed with status | `API error: status code` 408, 409, 5xx |
| timed out | `timed out`, `no answer within llm.timeout` |
| no answer within the limit | try ran past limit + sdd's extraction bound + 30s; sdd and its children killed |

- **Gave up → exit 1:** `... on all 3 tries; the draft did not cause it, so capture it again later`.
  Never edit the draft; run again later. On timeouts also `a long draft may need more than <limit>;
  run again with a longer --preflight-limit`: do that.
- **Routes** (`--preflight-route <model>=<layer>`, in order): primary gave up → each route one try,
  sdd gets `--model <model> --local-config <layer>`, first answering decides. Stdout opens
  `SDD_CAPTURE_ROUTE <model>` (or `primary`). All failed → `route <name>: <fault>: <last line>` each,
  then `... failed on every route; the draft did not cause it, so capture it again later`. Layer
  missing, unreadable, not yours, or group/other-readable → `route <model> skipped: <layer> <why>`,
  never read: holds an API key.
- **Never retried:** a `[high]` finding (even one quoting "timed out"), `validation failed`, status
  400/401/403/404.
- **Older schema** → ``the graph needs `sdd migrate` before sdd writes to it; the draft did not cause
  this``; stock sdd: ``sdd refuses to write this graph; the draft did not cause this``. `sdd migrate`
  rewrites the whole graph: owner's call.
- Capture step never retried: failure there = file or git, entry may be on disk. `git log` first.

## After

Exit 0 → draft deleted, `draft removed: <path>`: never recreate it. Kept with a line:

| `draft kept: <path> ...` | Do |
|---|---|
| `is tracked by git` | remove in its own commit, if wanted |
| `is outside the work tree` | may be another's: leave, or remove once sure |
| `is not a regular file` | link and target untouched |
| `changed while it was checked` | look before removing anything |

Kept silently: plugin option `capture_remove_draft` off, git could not tell, or the deletion
failed; delete once the entry reads right. Exit 1 or 2 → draft stays: fix it, run again.
