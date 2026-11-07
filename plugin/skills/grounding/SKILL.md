---
name: grounding
description: 'Grounding an sdd decision graph before asking, drafting or capturing: both search modes, the listing, read depths, ref kinds, every gate refusal. Use when reading or writing entries in an sdd graph, when the grounding gate refuses a call or lets one through with a note, before putting a question or editing a draft, and before any capture. Only projects carrying a .sdd/ graph.'
license: MIT
---

# Grounding

Agent asked about a graph answers from the entry it already read, or from what it believes the graph
holds. Both → wrong answer, written down, confident.

**Only a project carrying `.sdd/`.** No graph → gate silent, nothing here applies.

**Gate holds on stock sdd and the fork alike.** It reads commands only; every one it matches exists
in both.

**Gate on unless `.sdd/grounding.yaml` lifts it.** `enabled: false` → gate silent for every session.
Gate silent → the reads still keep answers right; nothing enforces them.

## Three reads, before any question, draft edit or capture

**Credit lasts until the graph changes or the context compacts.** Read once, work across prompts on
an unchanged graph. Another session's capture or a compaction → read again. Your own capture keeps
your reads while its entry is the only change.

```bash
sdd search --query "<how somebody would phrase it>" --limit 5 --max-citations 0   # semantic
sdd search --term  "<literal token: flag, command, id>" --limit 5 --max-citations 0
sdd view --layout "active:as-counts"                   # every topic, no prefix set
sdd view --layout "topic(<prefix><name>):as-list"      # one topic, prefix set
```

- **Five header lines per search** (`--limit 5 --max-citations 0`): enough to see what the graph
  holds on the subject. Broad subject → higher `--limit`. A body → `sdd show`. Any limit counts.
- **Which listing → `.sdd/grounding.yaml`.** `listing_prefix: area-` → one topic carrying it, whole.
  No file, no key → every topic. The refusal names the one owed.
- **Which command → `.sdd/grounding.yaml` too.** `read_command` → suggested reads use it in place of
  `sdd`; bare `sdd` still counts. `read_suffix` → printed at the end of every suggested read, never
  required: `python3 scripts/graph.py search --term x --entry <entry>`. Neither → bare `sdd`, as
  here. The refusal prints the project's form.
- **One call → `gather_command`.** Set → it runs all three: `--query` counts as semantic, `--term`
  as literal, each `--area <prefix><name>` as that listing, no `--area` as the `--entry` subject's
  areas. A flag left out → that mode still owed. The refusal prints the call.
- **`sdd-capture` → the reads standing, or a grounded draft.** Grounded = saved through
  `Write`/`Edit` under a `drafts/` directory while all three stood, unchanged since, graph holding
  the same entries. Draft written through `Bash`, edited after, or another capture landed between →
  run the three first.
- **Plain `sdd new` → the three reads, always.** No draft, nothing grounded. `--dry-run`, `--help`,
  `-h` exempt, on `sdd-capture` too. Wrapping (`bash -c`, `eval`, subshell, `xargs`, full path,
  any quoting) changes nothing. A command that does not parse and names `sdd` or `sdd-capture`
  owes the three too.
- **sdd's MCP tools count the same.** `mcp__sdd__search` `query` = semantic, `terms` = literal;
  `mcp__sdd__view` `layout` = the listing, same prefix rule. Only a call that succeeded counts.
  `mcp__sdd__start_procedure` with `canonical: capture` → the three reads; no grounded draft stands
  in.
- **`"shown_earlier": true` → text your context already holds, unchanged.** Read it where it printed
  earlier. A compaction, `/clear` or resume empties the record; text lost there prints in full again.
  Need it again → `mcp__sdd__show` that id alone (asked ids always print in full); a lane or schema
  → `mcp__sdd__resume_session`.
- **Both modes. Always.** Semantic matches phrasing, misses other wording. Literal matches the token,
  misses everything phrased differently. A live decision already owning your subject → exactly what
  one mode alone hides.
- **Empty result ≠ absence.** `sdd search` finds what it matched. `sdd show` reaches only what
  something cited. A listing → only read settling what a topic holds.
- **`sdd info` once per session**, before the first search. `vector` where an embedding endpoint
  answers, `text` otherwise.

## Reading an entry

**Every grounding read → `sdd show <id> --down 2 --up 1`.** Depths are part of the command, not a
default to weigh. `show_depth` in `.sdd/grounding.yaml` may owe another depth; the refusal echoes the
call at the depth owed: rerun it as printed. MCP → `mcp__sdd__show` with `"down": 2, "up": 1` spelled
out: an omitted `down` reads as 1 and is refused.

```yaml
show_depth:
  down: 3                    # left out → 2; 0 → nothing owed that way
  up: 1                      # left out → 1
```

- Body immutable → a surface it names keeps that spelling forever, after a rename too. Rename lives
  downstream, nowhere else.
- Status stays `active` throughout. Refinement does not close its target.
- `--down 2` → refinement of a refinement. `--up 1` → what the entry answered: says whether it still
  applies.
- **One exception: a `prc`-layer rules entry.** Downstream chain = every entry ever captured under
  it.

## Body = lead. Never ground truth.

0. **Search for a decision already owning it.** Active decision settles it → implementable, not
   open. Draft another → duplicate that disagrees the moment either changes. Marker: `addresses` on
   a live decision.
1. **Read the code it names, at the line it names.** `file:line` was true when written.
2. **Read the spec section under `docs/` governing it.** Entry describes code. Document says what
   code owes. They disagree more often than either admits.
3. **Verify every external claim against that party's own docs.** Provider limit, API absence, tool
   behaviour.

**Report which claims held, which did not.** Settled entry on a false premise ≠ one that was right.

| Failure shape | Reads as |
|---|---|
| Stale | tree moved, entry did not. Gap already closed |
| Wrong blocker | real defect named, governs another surface |
| Prose-only mechanism | function or recovery path exists only in the comment naming it |
| Wrong component | wrong binary or party. Check credential scope first |
| Already decided | body poses a question a live decision elsewhere answers |

## Ref kinds

Immutable. Get it right before capture. Full set: `sdd new --help`, under `--refs`.

| Kind | When |
|---|---|
| `refines` | narrows or corrects a **live** commitment, in place |
| `addresses` | supplies what an **active** target asked for |
| `builds-on` | forward step, or **closed** target |
| `grounded-in` | this entry reasons **from** the target |
| `required-by` | recorded from the prerequisite's side |
| `related` | floor kind. Understates every time. Pre-flight flags it |

## What the gate lets through, and what it refuses

**A question, a draft edit or a notebook edit under `drafts/` is never refused.** Reads missing →
the call goes ahead with a note opening ``GROUNDING GATE: let through, reads owed before the next
question or draft.`` A draft edit's note adds `This draft is not grounded: its capture owes the
reads.` The note names only the reads still owed, under `Owed since the graph last changed:`, each
label above its command. A read not named is credited: never re-run it. Run the named ones before the
next question or draft; never re-ask the call that went through. Question already asked → read
before answering what it settled. The note fires once per cause, the set of reads missing under the
current graph: later gated calls owing the same reads go ahead silently, so act on the first note.
It fires again after a graph change, a compaction, or a change in which reads are missing. A refusal
for reads owed opens ``GROUNDING GATE: refused. Run the reads owed, then run this call again.``, on
every call.

**Broken `.sdd/grounding.yaml`** → those calls go through as ``GROUNDING GATE: let through.`` with the
parse error; every other gated call but an `mcp__sdd__next` report refused, ``Fix `.sdd/grounding.yaml`, then run this call again.``
Fix the file.

| Refused | Why, and the fix |
|---|---|
| `sdd-capture` where `.mcp.json` declares `mcpServers.sdd`, no `cli_capture: true` | engine's capture plays the entry back and checks its summary: use `mcp__sdd__start_procedure` (`sdd-helpers:capture`) |
| capture, three reads missing, draft not grounded | writing what the graph never confirmed |
| `sdd new` without `--dry-run` or `--help`, three reads missing | same, by the path that skips the draft |
| engine capture start or report breaking the project's `listing_prefix` or `draft_lint`, or a `bodyPatch` or `bodyHandle` it holds no body for | sdd's guide and pre-flight never apply those rules. Each refusal and its fix: `sdd-helpers:capture`. The handle one: ``GROUNDING GATE: refused. This `bodyHandle` names `<handle>`, a staged file the gate holds no record of, so the project's draft lint (`draft_lint` in `.sdd/grounding.yaml`) cannot read the body.`` |
| `sdd` call sending a stream to `/dev/null` | wrong flag prints usage to stderr. Discarded → reads as an empty graph |
| `sdd show` short of the depth owed (`--down 2 --up 1` by default) | rename lives downstream; upstream says if it still applies |
| a write holding what looks like a credential: `sdd new`'s command or a file it `--attach`es, an `sdd-capture` draft or a file its `attach` names, an `mcp__sdd__next` report, `mcp__sdd__start_procedure` params, an `mcp__sdd__stage_attachment` file. `GROUNDING GATE: refused. The capture holds what looks like a credential (<rule ids>, <where>, line <n>); the graph is shared and immutable` | entry never edited, every session reads it. Reads or a grounded draft change nothing: take the credential out (name the variable or file holding it), capture again. `scan-incomplete` = scan could not finish |
| a write whose scan the hook's own interrupt cut short, ahead of every check but a credential found. `GROUNDING GATE: refused. The hook was cut short before its credential scan finished, and nothing passes unscanned; run the call again` | names no credential, no reads owed: the gate never saw the write whole. Run the same call again |
| a write the gate cannot scan, once nothing else refuses it. `GROUNDING GATE: refused. The capture names <file>, which cannot be read to scan for a credential; it must exist before the Bash call that captures it` / `GROUNDING GATE: refused. The capture attaches standard input as <name>, which cannot be scanned for a credential; attach a file instead` / `GROUNDING GATE: refused. The capture names no draft that can be read to scan for a credential; name the draft file in the call` / `GROUNDING GATE: refused. The capture names <file>, which cannot be read to scan for a credential: <why>; attach it by a literal path, relative to the project directory with no cd before it` / `GROUNDING GATE: refused. The <tool> call's input cannot be read to scan for a credential` | write the draft or file in a call of its own first; attach a file, not `-:<name>`; name the draft in the `sdd-capture` call; attach by literal path, relative to the project directory, no `cd` or launcher in the call. `<why>`: `it holds a shell expansion` (`$`, `~`); `it holds a shell glob or brace` (unquoted `*`, `?`, `[`, `@(...)`, `{a,b}` in any argument of the `sdd new` or `sdd-capture` call, or any patterned command word or `sdd` subcommand in a call naming `sdd`, `new` or `capture`, as `sd? new`, `{,}sdd new` or `sdd n?w`, named as `<file>`: quote it or name each file); `it is relative after a directory change`; `it is relative behind a launcher` |

**No binary installed** → `sdd-helpers: no sdd-grounding-gate binary is installed, so the grounding
gate refuses every call; ...`, then the fix: a session with network fetching the release, or
`SDD_HELPERS_BIN_DIR`. The user's to fix, never yours.

## Traps

- **`--term`, repeatable. Not `--terms`.** Wrong flag → usage on stderr, non-zero exit.
- **Never `2>/dev/null` an `sdd` call.** Filter instead: `| grep -E '^  [0-9]'`.
- **A listing is enormous.** Another capture mid-pass invalidates it. Read the first run whole, then
  only what changed.
- **Which capture path, the draft, what an entry carries** → `sdd-helpers:capture`.

## Off on the machine

`grounding_gate_enabled: false` → gate lets every call through, keeps no evidence, and
`sdd-capture` goes off with it (`sdd-helpers:capture`). Run the reads anyway; capture is the
user's to switch back on.
