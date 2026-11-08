---
name: capture
description: 'Capturing an sdd graph entry: through sdd''s engine (mcp__sdd__start_procedure, canonical capture) wherever mcp__sdd__* tools are present, through sdd-capture from a draft file only where they are not. What an entry carries, the project''s rules on an engine capture, every refusal. Use before any capture, and whenever sdd-capture or the grounding gate refuses one. Only projects carrying a .sdd/ graph.'
license: MIT
---

# Capture

## Which way

| Session has | Capture through |
|---|---|
| `mcp__sdd__*` tools (sdd's engine) | `mcp__sdd__start_procedure`, `canonical: capture`, driven by the `sdd` skill. Never `sdd-capture` |
| no sdd MCP server, or one refusing the graph | `sdd-capture <draft.md>`: [sdd-capture](references/sdd-capture.md) |

- `.mcp.json` declares `mcpServers.sdd` → gate refuses `sdd-capture`:
  `GROUNDING GATE: refused. This project serves sdd's engine ...`. `cli_capture: true` in
  `.sdd/grounding.yaml` lifts it: owner's setting, never yours. A server outside `.mcp.json` goes
  unseen.
- Both paths owe the grounding reads (`sdd-helpers:grounding`).
- Never a credential in an entry: name the variable or file holding it.

## What an entry carries

- **What, why, rejected alternatives with their reasoning, accepted costs.** No rejected
  alternatives → re-proposed within the month.
- **Every surface named exactly:** command, flag, RPC verb, config key, label, path, environment
  variable. "We'll add a command" → not settled.
- No acceptance criterion waits on a decision the entry could take itself.
- Plain prose, people read it. Body never fenced: a lint reads fenced text as code.
- At least one topic; `listing_prefix` set → one carrying it.
- Ref kinds: `sdd-helpers:grounding`. Immutable: right before capture.
- After: read the generated summary. Drift (wrong actor, ref attribution, lost identifier) → engine
  `verifySummary` `drifted` with `correctedSummary`; otherwise `sdd summarize <id> --text '<summary>'`.

## Engine capture: the project's rules

sdd's guide and pre-flight never apply `.sdd/grounding.yaml`. The gate does, on the capture's start
and every `mcp__sdd__next` to it (`params`, `report`, a chooser's `fields`).

| Rule | Refused as | Fix |
|---|---|---|
| `listing_prefix` | ``GROUNDING GATE: refused. These `topics` carry none opening with `<prefix>` ...``; playback `confirm` while none held: ``... This capture holds no topic opening with `<prefix>` ...`` | report `topics` with one `<prefix>` topic; at playback, `adjust` |
| `draft_lint` | ``... The project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) reports on the body this call leaves``, findings named `body:<line>:...` | fix each, report again |
| lint failing silent | ``... did not read the body this call leaves: <why>`` | make the lint run, call again |
| `bodyPatch`, no body held | ``... This `bodyPatch` edits a body the gate holds no record of`` | report `body` in full once |
| `bodyHandle`, no staging seen | below | stage again, or report `body` |

- `bodyPatch` → applied to the last whole body by sdd's rule, result linted.
- `bodyHandle` → the gate lints the text it recorded at `mcp__sdd__stage_attachment` (`content`, the
  `path` file, `patches` applied), leading `---` header block left out. Staged before the gate saw it,
  or ledger lost → asked once per handle:
  ``GROUNDING GATE: refused. This `bodyHandle` names `<handle>`, a staged file the gate holds no record of, so the project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) cannot read the body.``
- Engine pre-flight fails `claude -p timed out` or `no answer within llm.timeout` → only sdd's
  `llm.timeout` reaches it; raising it (`sdd config set --local llm.timeout <duration>`) is the
  owner's call: tell them.

## Off on the machine

`sdd-capture: off on this machine (plugin option capture_enabled)` or
`sdd-capture: off while grounding gate is off (plugin option grounding_gate_enabled)`: exit 1,
nothing runs. Tell the user; never `sdd new` instead.
