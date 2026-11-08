# Grounding gate refusals

Reads owed: [SKILL.md](../SKILL.md#gate-notes). Every other refusal:

| Refused | Why, and the fix |
|---|---|
| `sdd-capture` where `.mcp.json` declares `mcpServers.sdd`, no `cli_capture: true` | engine's capture plays the entry back and checks its summary: use `mcp__sdd__start_procedure` (`sdd-helpers:capture`) |
| capture, three reads missing, draft not grounded | writing what the graph never confirmed |
| `sdd new` without `--dry-run` or `--help`, three reads missing | same, by the path that skips the draft |
| engine capture breaking `listing_prefix` or `draft_lint`, or a `bodyPatch` or `bodyHandle` it holds no body for | sdd's guide and pre-flight never apply those rules; each refusal and fix: `sdd-helpers:capture`. The handle one: ``GROUNDING GATE: refused. This `bodyHandle` names `<handle>`, a staged file the gate holds no record of, so the project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) cannot read the body.`` |
| `sdd` call sending a stream to `/dev/null` | wrong flag prints usage to stderr; discarded, it reads as an empty graph |
| `sdd show` short of the depth owed | rename lives downstream; upstream says if it still applies |

## Credentials

Scanned: `sdd new`'s command and every file it `--attach`es, an `sdd-capture` draft and every file
its `attach` names, an `mcp__sdd__next` report, `mcp__sdd__start_procedure` params, an
`mcp__sdd__stage_attachment` file. An entry is never edited and every session reads it, so reads or
a grounded draft change nothing.

| Refusal | Fix |
|---|---|
| `GROUNDING GATE: refused. The capture holds what looks like a credential (<rule ids>, <where>, line <n>); the graph is shared and immutable` | take it out, name the variable or file holding it, capture again. `scan-incomplete` = scan could not finish |
| `GROUNDING GATE: refused. The hook was cut short before its credential scan finished, and nothing passes unscanned; run the call again` | run the same call again; nothing owed |
| `GROUNDING GATE: refused. The capture names <file>, which cannot be read to scan for a credential; it must exist before the Bash call that captures it` | write the file in a call of its own first |
| `GROUNDING GATE: refused. The capture attaches standard input as <name>, which cannot be scanned for a credential; attach a file instead` | attach a file, never `-:<name>` |
| `GROUNDING GATE: refused. The capture names no draft that can be read to scan for a credential; name the draft file in the call` | name the draft in the `sdd-capture` call |
| `GROUNDING GATE: refused. The capture names <file>, which cannot be read to scan for a credential: <why>; attach it by a literal path, relative to the project directory with no cd before it` | literal path, relative to the project directory, no `cd` or launcher in the call |
| `GROUNDING GATE: refused. The <tool> call's input cannot be read to scan for a credential` | the engine call's input is no JSON object, so it cannot be scanned: call again with object input |

`<why>` reads:
- `it holds a shell expansion`: `$`, `~`.
- `it holds a shell glob or brace`: unquoted `*`, `?`, `[`, `@(...)`, `{a,b}` in any argument of
  the `sdd new` or `sdd-capture` call, or a patterned command word or `sdd` subcommand in a call
  naming `sdd`, `new` or `capture` (`sd? new`, `{,}sdd new`, `sdd n?w`). Quote it, or name each
  file.
- `it is relative after a directory change`.
- `it is relative behind a launcher`.

## Setup

- **Broken `.sdd/grounding.yaml`** → questions and draft edits go through as
  ``GROUNDING GATE: let through.`` with the parse error; every other gated call but an
  `mcp__sdd__next` report refused, ``Fix `.sdd/grounding.yaml`, then run this call again.`` Fix the
  file.
- **No binary** → `sdd-helpers: no sdd-grounding-gate binary is installed, so the grounding gate
  refuses every call; ...`, then the fix: a session with network fetching the release, or
  `SDD_HELPERS_BIN_DIR`. The user's to fix, never yours.
