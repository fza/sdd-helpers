---
name: grounding
description: 'Grounding an sdd decision graph before asking, drafting or capturing: the three reads, read depths, ref kinds, the gate''s notes and refusals. Use when reading or writing entries of an sdd graph, when the grounding gate refuses a call or lets one through with a note, and before any question, draft edit or capture. Only projects carrying a .sdd/ graph.'
license: MIT
---

# Grounding

Answer from belief, or from the one entry read → wrong, written down, confident. Gate holds
questions, draft edits and captures until the graph was read. Off: `enabled: false` in
`.sdd/grounding.yaml` (project), plugin option `grounding_gate_enabled: false` (machine,
`sdd-capture` off with it). Off → reads still owed, nothing enforces.

sdd-helpers is deprecated. Session start says `deprecated and standing aside` → the fza/sdd fork
gates captures itself: this gate is off, follow the `sdd` skill, reads through `gather`.

## Three reads

Before every question, draft edit, capture. Credit holds until the graph changes or context
compacts; own capture keeps it.

```bash
sdd info                                                  # once per session: vector or text search
sdd search --query "<how somebody would phrase it>" --limit 5 --max-citations 0   # semantic
sdd search --term "<literal: flag, command, id>" --limit 5 --max-citations 0      # literal
sdd view --layout "active:as-counts"                      # listing: every topic
sdd view --layout "topic(<prefix><name>):as-list"         # listing: listing_prefix set
```

- **Both search modes, always.** Each misses what the other finds. Empty result ≠ absence; only a
  listing settles what a topic holds. Any `--limit` counts.
- **MCP counts the same:** `mcp__sdd__search` `query` / `terms`, `mcp__sdd__view` `layout`.
  Successful calls only.
- **Note or refusal prints the project's form** (`read_command`, `gather_command`, prefix): run it as
  printed. Keys: [config](references/config.md).
- `--term`, never `--terms`. Never `2>/dev/null` an sdd call: a wrong flag prints usage to stderr,
  read as an empty graph. Filter instead: `| grep -E '^  [0-9]'`.

Owed by (`--dry-run`, `--help`, `-h` exempt):
- `sdd new`, however wrapped → three reads.
- `sdd-capture` → three reads standing, or a grounded draft: saved by `Write`/`Edit` under `drafts/`
  while they stood, unchanged since, graph unchanged.
- `mcp__sdd__start_procedure` `canonical: capture` → three reads. No draft stands in.
- How to capture: `sdd-helpers:capture`.

## Reading an entry

`sdd show <id> --down 2 --up 1`. MCP: `mcp__sdd__show` with `"down": 2, "up": 1` spelled out; an
omitted `down` reads as 1, refused. `show_depth` may owe more; the refusal echoes the call: rerun it
as printed. Only a `prc`-layer rules entry reads shallower.

- Down: renames and refinements. A body never changes and keeps its old spelling.
- Up: what it answered, whether it still applies. Refined entries stay `active`.
- `"shown_earlier": true` → text already in context, unchanged: read it there. Lost (compaction,
  `/clear`, resume) → `mcp__sdd__show` that id alone; lane or schema → `mcp__sdd__resume_session`.

## Body = lead, never truth

0. **Decision already owns it?** (`addresses` on a live decision) → implementable, not open. Never
   draft a duplicate.
1. **Code it names**, at the line named.
2. **`docs/` section governing it.**
3. **External claims** against that party's own docs.

Report which claims held. Failure shapes: stale (tree moved), wrong blocker, prose-only mechanism,
wrong component, already decided.

## Ref kinds

Immutable. Full set: `sdd new --help`, `--refs`.

| Kind | When |
|---|---|
| `refines` | narrows a **live** commitment in place |
| `addresses` | supplies what an **active** target asked for |
| `builds-on` | next step, or **closed** target |
| `grounded-in` | reasons **from** the target |
| `required-by` | from the prerequisite's side |
| `related` | floor kind. Understates. Pre-flight flags it |

## Gate notes

Question, draft edit, notebook edit under `drafts/`: never refused. Reads missing → goes ahead with
``GROUNDING GATE: let through, reads owed before the next question or draft.`` Draft edit adds
`This draft is not grounded: its capture owes the reads.` Lists only the reads owed, under
`Owed since the graph last changed:`; a read not named is credited, never rerun it. Run the named
ones before the next question or draft. Question already asked → read before answering it.

Note fires once per cause: later calls owing the same reads pass silent, so act on the first. Fires
again on a graph change, a compaction, or a new set of reads owed.

Other gated calls → ``GROUNDING GATE: refused. Run the reads owed, then run this call again.``
Every other refusal and its fix: [refusals](references/refusals.md).
