# The grounding gate: architecture

`sdd-grounding-gate` runs in a project carrying `.sdd/`, at the project root or above it, and does
nothing elsewhere or where `.sdd/grounding.yaml` says `enabled: false`.

## Which graph

The hooks find a project's graph where sdd does. They walk up to the governing `.sdd/` and read its
`config.yaml`, overlaid by `config.local.yaml`. Where that sets `project_dir`, sdd's split layout,
the code repository's `.sdd/` holds only that key, and the project is the graph repository it names:
its `.sdd/` holds `grounding.yaml`, the gate's ledgers and `sdd-capture`'s lock. Entries live under
that project's `graph_dir`, `.sdd/graph` where unset. A relative `project_dir` or `graph_dir` joins
to the folder holding the `.sdd/` that sets it, never to the git toplevel. A `graph_dir` leaving that
folder reads as unset: sdd refuses it, and the split layout names its graph repository through
`project_dir` instead. A `project_dir` naming no `.sdd/` keeps the code repository's own `.sdd/` and
reads an empty graph, creating nothing in the missing checkout.

## Modes

`sdd-grounding-gate <mode>` reads the hook payload on standard input. Every mode but `check` and
`prune` exits 0 at once where `grounding_gate_enabled` is off or no graph governs the project.

| Event | Mode | What it does |
|---|---|---|
| SessionStart | `start` | removes the ledgers of sessions that left a week ago without ending cleanly; with `source: compact`, forgets this session's reads and keeps its grounded drafts; switches collapsing on or off for the session (see [Collapsing](#collapsing)); tells a failed release prune once and starts the day's prune, detached (`internal/releaseprune`, [plugin architecture](../plugin/architecture.md#release-pruning)) |
| SessionStart | `check` | prints the plugin's session-start lines ([plugin usage](../plugin/usage.md#what-each-session-start-says)); tells a failed release prune once and starts the day's prune, detached, as `start` does; runs whether or not the gate is on, in every project |
| PostToolUse on `Bash`, `mcp__sdd__search`, `mcp__sdd__view`, `mcp__sdd__start_procedure`, `mcp__sdd__next` | `record` | credits the reads the call performed; after the session's own capture (`sdd-capture` printing `<id>.md`, or a `next` result carrying `produced.entryId`), moves the ledger to the new graph state when that entry is the only change, keeping the reads; keeps the body and topics an engine capture's start or report left it with (`recordEngineCapture`) |
| PostToolUse on `mcp__sdd__stage_attachment` | `record` | where `.sdd/grounding.yaml` sets `draft_lint`: keeps the text the staging left under the handle sdd answered with (`recordStaged`) |
| PostToolUse on `Write`, `Edit` | `record` | grounds a draft saved under a `drafts/` (or `draft/`) directory while every read stands, hashing the file as saved |
| PreToolUse on `AskUserQuestion`, `Write`, `Edit`, `NotebookEdit`, `Bash`, `Monitor` (its command read as a Bash command), `mcp__sdd__show`, `mcp__sdd__start_procedure`, `mcp__sdd__next`, `mcp__sdd__stage_attachment` | `gate` | for `AskUserQuestion`, `Write`, `Edit` and `NotebookEdit` (by `file_path` or `notebook_path` under `drafts/`), exits 0 with the refusal's text as `hookSpecificOutput.additionalContext`, its first line saying the call went ahead and, for reads owed, that they are owed before the next question or draft (`letThrough`); for every other call, refuses with exit 2 and the refusal on stderr; or passes with exit 0 and nothing |
| PostToolUse on `mcp__sdd__.*` | `collapse` | replaces an answer carrying sdd's structured fields with its collapsed form, as a plain JSON string under `hookSpecificOutput.updatedToolOutput` (`internal/collapse`), and marks what it printed in full in the calling agent's set |
| PreToolUse on `Agent`, `Task` | `spawn` | for `subagent_type: fork`, keeps the spawner's set under the call's `tool_use_id` |
| SessionEnd | `end` | removes the session's ledger and its shown sets |
| none: `start` and `check` run it detached, at most once a day | `prune` | deletes the plugin's release folders no session may still run, prints `deleted <folder>` for each, and records the run in `<data>/release-prune.json`; exits 1 on stderr where the run failed; reads no payload and works whatever the project, and touches nothing without `CLAUDE_PLUGIN_DATA` and `CLAUDE_PLUGIN_ROOT` |

## The ledger

A ledger's reads carry `graph`, the graph's state when they were recorded: the count of entry files
under the resolved `graph_dir` and the newest entry's name, the state grounded drafts carry too.
`record` compares it before crediting a read, and a different state clears the query, term and
listings first; `gate` treats reads recorded under another state as none.

The gate keeps each session's credited reads and grounded drafts under
`.sdd/tmp/grounding-gate/`, one ledger per session. A ledger is replaced through a temporary file of
its writer's own, renamed over it (`internal/atomicfile`), each folder from the one holding the graph
down walked through `internal/safefs` ([plugin architecture](../plugin/architecture.md#writes)), so
a `.sdd/tmp` committed as a link is never written through, and the parallel hooks of one call never
leave it half written; a torn ledger would refuse reads the session made. A ledger that cannot be
written holds the reads nowhere, so the gate refuses, the safe side. A grounded draft is recorded
with the graph's entry count and newest entry at the time it was saved; a capture of it passes only
while the draft file is byte-identical and the graph still holds the same entries.

`captures` holds each engine capture the session started, by its instance handle: the last whole
body and the topics the engine holds for it, and whether the gate already asked for the whole body
once. The serve a start answers with names the instance; the gate reads it whether the client hands
on the serve's JSON as one string (as Claude Code does), the serve itself, the MCP result's
`structuredContent`, or JSON in a text block. Only a call that went through is recorded, a
`bodyPatch` applied by sdd's own rule (`applyPatch`), and a serve reporting the capture `completed`
or `abandoned` drops it. Neither a graph change nor a compaction clears `captures`: it mirrors the
engine's state, not a read.

`staged` holds, where the project sets `draft_lint`, the text of each file the session staged
through `mcp__sdd__stage_attachment`, by the handle sdd answered with, read in each form a serve
is (`answerOf`). A staging that went through records the `content` given, the file a `path` names
(`readNamed`, as the credential scan reads it), or the held text with the `patches` applied by
`applyPatch`: sdd edits the file staged under `name` in place and answers with its handle. Text
whose sha256 differs from the `sha256` sdd answered with, a patch the gate cannot apply, or a path
it cannot read drops the record: the gate does not know the file. A failed staging, or an
answer naming no handle, changes nothing. An entry with no text marks a handle already refused
once. A ledger written without `staged` reads as holding none, and every key the ledger carries
survives a record. Like `captures`, `staged` mirrors the engine's state and neither a graph change
nor a compaction clears it.

`noted` holds the missing reads the last let-through note named (`firstNote`). A gated call whose
missing reads match it gets no note. A different graph state clears it with the reads, a compaction
clears it with them, and a different set of missing reads does not match, so each of the three
fires the note again. A ledger that cannot be written leaves the note firing.

## The reads refusal

`refusals/reads.txt` holds the frame: its first line, the `gather_command` call where the project
names one, and the reads owed. `owedReads` fills the last from the ledger: one header, then a label
and command for each read the ledger does not credit, and appends the search hint only when a search
is among them. A read the ledger credits is never named, so an agent is not sent to repeat it. On a
gated tool `letThrough` swaps the first line for the let-through one: the call already ran, so the
reads are owed before the next question or draft rather than before the call again. The note
carrying reads owed fires once per cause (see [the ledger](#the-ledger)); a refusal fires on every
call.

## The engine capture checks

`engineCaptureRefusal` holds a capture start, once its reads stand, and every `mcp__sdd__next` to an
instance in `captures`, to the project's `listing_prefix` and `draft_lint`. It reads the start's
`params` or the report, with a chooser answer's `fields` folded in. Topics lacking the prefix refuse
(`refusals/engine-topics.txt`), and so does a `confirm` to the `playback` chooser while the held
topics lack it (`engine-confirm.txt`). The body a call leaves, a whole `body` or a `bodyPatch`
applied to the held body, goes through `internal/draftlint`, the runner `sdd-capture` shares:
written to `.body-*.md` in the ledger directory, run from the project's root, each finding named with
the file as `body` (`engine-lint.txt`); a run that fails silently refuses (`engine-lint-unread.txt`).
A patch with no held body refuses once (`engine-body.txt`) and passes after. The capture procedure's
field names, its `playback` step and sdd's edit rule are restated here, so a change to either in sdd
needs one here.

A call naming `bodyHandle` has the text held under that handle in `staged` linted the same way,
after a leading header block opened and closed by a `---` line (`stagedBody`, sdd's
`SplitStagedBody` restated: a leading byte order mark dropped, the body everything after the
closing line, a file with no closing line all body). A handle with no text held refuses once
(`engine-handle.txt`) and passes after. A report naming the handle also leaves its body as the
capture's held body, the one a later `bodyPatch` edits once the capture drops the handle.

## The engine refusal

On `Bash`, a call running `sdd-capture`, recognised in every spelling the gate sees a capture in, is
refused with `refusals/engine.txt` when the project's `.mcp.json`, read from the project root,
declares `mcpServers.sdd`, the server name sdd's `mcp__sdd__*` tools carry, and the resolved
`.sdd/grounding.yaml` sets no `cli_capture: true`. Where the governing `.sdd/` names its graph
repository through `project_dir`, the server counts only where the installed sdd lists
`--project-dir` in `sdd init --help`: an sdd without it ignores the key, and its server would serve
the code repository's own `.sdd/`. The probe's answer is cached per binary
([plugin usage](../plugin/usage.md#sdd)). The check runs before the reads: credit decides nothing
here. A `.mcp.json` that is missing or does not parse declares no server. An sdd server at user or
local scope is not read.

## Crediting a read

A command is parsed as bash with `mvdan.cc/sh/v3/syntax`, and `internal/shellwalk` lists every
command it runs: in a subshell, a pipe or a command substitution, after a launcher (`env`, `xargs`,
`sudo`, `find -exec`, `xcrun`, `uv run`, ...) past the launcher's options, assignments and counts,
and inside code a shell runs (`bash -c`, `eval`, `trap`, a here-document fed to a shell or `at`,
`sched`, a tmux shell command, zsh's `functions[name]=`, text arithmetic evaluates), and inside a
shell script it runs (`bash x.sh`, `source x.sh`, `./x`), read through `shellwalk.WalkWith` from the
hook's working directory, so an `sdd new` inside a sourced script owes the reads any write owes. A
script the walk cannot read is not judged: no gate can say what such a script writes. A Monitor
call's command is read as a Bash command's. An option's value that is neither fixed text without a
glob or brace nor wholly double-quoted without an `@`, `*` or `!` expansion (`sudo -u $user sdd ...`)
is taken for the command, as it may split into one, so a read behind it is not credited. A quoted
argument and a heredoc body are words, never a call, so crediting a quoted mention cannot let an
`echo` satisfy the gate. Each call's words are read with their quoting removed (`$'...'` decoded, a
backslash inside double quotes dropped), and the search mode, layout and depth are read from those
words. A command's first word counts by its base name; a `read_command` or `gather_command` is
matched word for word after it, so each must be plain words without quotes, variables or operators.

Only the calls the command itself runs are credited: a read inside code a shell runs is checked for
a discarded stream and its depth, but credits nothing. A stream counts as discarded where a
redirection in the same stretch of code (the command, or one piece of code a shell inside it runs)
sends output to `/dev/null`.

A command that does not parse is read from its text: wherever `sdd` or `sdd-capture` stands in it as
a word, it counts as a plain `sdd new`, so its reads stay owed, and a discarded stream or a short
show in its text is refused. It credits nothing.

## Collapsing

The `start` mode asks `sdd config get sessions.token_saving` in the graph's repository and writes
`.sdd/tmp/grounding-gate/<session>.collapse-on.json` where it prints exactly `off`, removing it
otherwise; `collapse` and `spawn` do nothing for a session without it, since each runs on every sdd
call and starts no process of its own. Under `on`, sdd stubs repeats itself, and the two would
rewrite one answer twice. The marker, and emptying the set at a compaction, `/clear` and resume, run
before the `.sdd/grounding.yaml` check, so a project lifting the gate keeps both.

`internal/collapse` reads only the fields sdd answers under `sessions.token_saving: off`: `items`
(show: entries, each with `upstream` and `downstream` as preorder lists carrying `depth`), `hits`
(search), `view_sections` (view), and `lanes` and `framing_blocks` beside `report_schema` (every
serve, the ones under `open_instances` and `base_junction` too). An answer without any of them is
never rewritten, so no text layout is parsed and an older sdd's answers pass whole. Which of them
is read follows the tool, never a version: a `view` answer is read only through `view_sections`,
so a section's `list.items` is never taken for a show's `items`.

Each view section sets exactly one shape object, and the walk visits the entry objects at fixed
paths below it: `list.items[].entry`, `grouped.groups[].entries[]`, `focus.focuses[].focus`,
`focus.focuses[].targets[].target`, `participants.groups[].actor` and `wip.markers[].entry`. An
entry whose line the agent holds becomes `{id, title, shown_earlier: true}` in place, so every
field beside it (`score`, `refs`, `actors`, `when`, `state`, `roles`, a marker's own `id` and
`content`) stays. `bodies.entries[]` only mark their lines, and `counts.rows` hold no entry.
Visiting fixed paths rather than every object holding an `id` matters: a ref and a wip marker
carry ids that name no line the view printed. The answer is accepted as an object, the object
encoded as a string, `structuredContent`, or the text of a content block, as Claude Code hands MCP
answers on.

The shortened answer goes back as one JSON object encoded as a plain string under
`hookSpecificOutput.updatedToolOutput`: the form Claude Code hands the agent an untouched sdd
answer in, since a replacement must match the tool's output shape. An MCP content wrapper under
`updatedMCPToolOutput` reaches the agent serialized, one escaped string in which every quote and
line break gains a backslash, longer than an untouched answer wherever little collapses. Every
piece is encoded with `SetEscapeHTML(false)` (`encode`), so `<`, `>` and `&` stay as sdd sent them
rather than turning into `\u003c` escapes the agent pays for and gains nothing from.

A set, `.sdd/tmp/grounding-gate/<session>.shown.json` (`<session>.<agent>.shown.json` for a
subagent), holds per entry id a digest of its title, kind, status and summary, and per served piece
(`lane:<name>`, `framing:<name>`, `schema:<digest>`) a digest of its bytes. A walk marks what an
answer prints in full only once the whole answer is read, so nothing collapses against the answer
printing it; an item sdd marks `seen_in_response` is left as sdd gave it. One lock file,
`collapse.lock`, serialises every read and write of the folder's sets, so parallel tool calls
neither lose a mark nor read a half-written set.

A fork's set starts from what `spawn` kept under `<session>.spawn-<tool_use_id>.shown.json`, found
through the `toolUseId` Claude Code writes into the fork's `agent-<id>.meta.json`, and the kept copy
is removed then. Keeping it at the spawn, not at the fork's first answer, matters: the spawner's later
reads are not in the fork's context. Any failure (an unreadable set, a lock refused, an answer that
does not decode) leaves the answer as sdd gave it.

## Credentials

`gate` scans a graph write before anything else (`internal/grounding/credentials.go`). A Bash call
writes where the parse tree holds an `sdd-capture` or `sdd new` call, or code that does not parse
names either; unlike the reads, a write into another repository's graph is scanned too. What it
scans:

- **The command's whole text.**
- **A plain `sdd new`'s `--attach` files**, in both the `--attach value` and `--attach=value`
  forms, read against the working directory.
- **An `sdd-capture` draft**: the call's draft argument (its first word that is no flag or flag
  value, `~` and variables expanded), or, where the call names none (a draft handed over through
  `xargs`), every `.md` name in the command; read against the working directory, else the project.
  Then every file its header's `attach` names, read from the directory holding the governing
  `.sdd/`, where sdd-capture runs sdd.
- **An attach value** splits as sdd splits it, on the last colon: `source:target` reads `source`;
  `-:target` reads standard input. Every attached or staged file is scanned by its name in the
  entry, so path rules fire, and where that finds nothing, as nameless text, since gitleaks' global
  allowlist skips whole files by name (`.bin`, images, fonts, office files) and
  `secret.txt:secret.bin` would otherwise pass unscanned. Only a literal path is read, never
  expanded: an absolute one, or a relative one in a command with no directory change anywhere and
  no launcher in front of the call (the launchers and their chdir options are `shellwalk.Options`,
  the table the walk reads too). Anything a launcher or an expansion would change is refused, not
  resolved. A plain `sdd new` or `sdd-capture` call is refused outright where any argument is a
  glob or brace the shell expands into other words, since the words the gate reads are spelled, not
  expanded: an attachment it never read, or an `--attach` it never saw, could reach sdd. That is
  decided on the parsed word (`shellwalk.Patterned`), not its text: an unquoted, unescaped `*` or
  `?`, an unquoted, unescaped `[` with a `]` after it in the same word (a lone `[`, as the test
  command `[`, is text in bash), an extended glob, or a brace expansion; quoted or escaped, the same
  characters are text. A call the gate would never classify as a write is refused the same way,
  before classifying, where its command word, or the subcommand of a call naming `sdd`, is
  patterned at all and any word of the call names `new` or holds `sdd` or `capture`, lower-cased
  and stripped of `[]{},*?`. The gate does not match the pattern: bracket classes (`[=d=]`,
  `[.d.]`), case folding inside brackets and brace alternatives that shift every word along
  (`{,}sdd new`) each read differently in bash from what a matcher would guess, so naming a write
  tool anywhere in such a call is enough. Code the parser cannot read (`sd[[=d=]] new`, which bash
  runs and mvdan.cc/sh rejects) is refused where it holds `*`, `?`, `[` or `{` and names a write
  tool the same way. A call behind a launcher counts as `callsCapture` counts one: certain, or
  handed an argument.
- **A call behind a launcher's options** the walk cannot place (`sudo -iD dir sdd new`,
  `flock lockfile sdd-capture d.md`) still counts as a write: `sdd new`, or `sdd-capture` handed an
  argument. It credits and checks no read.
- **For `mcp__sdd__next` the `report`, for `mcp__sdd__start_procedure` the `params`**, decoded,
  every key and string in them one per line in key order: the JSON text itself is not scanned,
  since its escapes hide a token from the rules. The session `label` and the other fields naming
  the dialogue are not scanned: they never reach the graph.
- **`mcp__sdd__stage_attachment`**: every key and string of the call, one per line in key order (its
  inline `content` and each patch's `old` and `new` text among them), and the local file its `path`
  names, read against the session's working directory; all by the staged `name`. A patch is scanned
  as the text it writes, not the file it leaves (see [what the scanner guards
  against](#what-the-scanner-guards-against)).

A finding refuses with `refusals/credential.txt`, naming the rules on the first line found and that
line (`secrets.Describe`), never the matched text, ahead of every other check. A `scan-incomplete`
finding refuses like any other, except once the hook's context has ended: the scan it cut short
found no credential, so any finding then refuses with `refusals/credential-interrupted.txt`, after
a credential an earlier scan found and ahead of every other check, since under an ended context the
ledger reads as empty and the reads check would name reads owed that are not.

The gate fails closed: what it cannot scan refuses. Standard input (`refusals/credential-stdin.txt`),
a file that cannot be read or a draft header that does not parse
(`refusals/credential-unreadable.txt`), a capture naming no draft
(`refusals/credential-no-draft.txt`), a write call with any argument holding a shell glob or brace,
an attachment path it will not place, holding `$` or `~`, or relative after a directory change or
behind a launcher (`refusals/credential-unresolved.txt`, naming why), and an engine input that is
no JSON object (`refusals/credential-undecodable.txt`) refuse once no other check does, so a turn
still owing reads hears that first. A credential anywhere in the write outranks what could not be
scanned.

The gate's scan is the early refusal: it reads files before the Bash call runs, so a draft written
by the same command is unreadable to it, and a file changed afterwards escapes it. `sdd-capture`
closes that gap by scanning the exact bytes it hands sdd right before each `sdd new`
([capture](../capture/architecture.md#credentials)); a plain `sdd new` and the engine have no
second scan.

## The credential scanner

`internal/secrets` is the one scanner the gate and `sdd-capture` share. Its rules are gitleaks'
default rule set, pinned: gitleaks v8.30.1, commit `83d9cd684c87d95d656c1458ef04895a7f1cbd8e`,
`config/gitleaks.toml`, provider rules for keys and tokens (AWS, GitHub, Slack, Stripe, private
keys, generic API keys and others) and one global allowlist. They are vendored as
`internal/secrets/rules.json`, embedded with `go:embed`, with gitleaks' MIT notice beside them as
`LICENSE-gitleaks`. `internal/secrets/vendor-rules.py` converts the TOML with Python's `tomllib`, so
the module carries no TOML dependency:

    python3 vendor-rules.py <gitleaks.toml> <version> <commit>

It writes `rules.json` beside itself; refreshing the rules means moving the pin, rerunning it and
committing the result. The sdd fork vendors the same pin with the same converter, so both refuse the
same strings. `internal/secrets/rules-local.json`, in the same rule shape, adds what the pin lacks:
Context7, Figma personal access, Melious, Mistral and Opper keys. They sit in a file of their own
because `rules.json` must stay the vendored file byte for byte; the sdd fork's scanner does not
refuse those strings.

`secrets.Scan(ctx, text, name)` returns the findings, each a rule id and the lines the match starts
and ends on, mirroring gitleaks' `detectRule` and `checkFindingAllowed` with Go's `regexp`:

- **Keywords gate a rule.** A rule with keywords runs only when one occurs in the lowercased text,
  and its patterns compile on first use. A rule without keywords always runs.
- **Path rules need a name.** `name` is the file the text is the content of, and empty for text
  that is no file (a command, a report). A rule with a `path` applies only to a named file whose
  name it matches; an allowlist naming the file skips it whole.
- **The secret** is the match's `secretGroup` group where the rule sets one, else its first
  non-empty capture group, else the whole match. A secret at or below the rule's `entropy` floor is
  skipped.
- **Allowlists** run global first, then the rule's own, against the secret, the match or the line,
  as the rule's `regexTarget` says.
- **Not done**: the inline `gitleaks:allow` marker is ignored, since nothing overrides a refusal;
  encoded segments (base64, hex, percent) are not decoded.

A scan that cannot finish, the context ended or a vendored pattern failed to compile, returns one
finding of rule `scan-incomplete`, so a caller refusing on any finding refuses rather than passes
unscanned text. The vendored rules compile in a test, so in a released binary only an ended context
reaches it, and a caller whose context ended reports the scan as cut short rather than as a
credential.

### What the scanner guards against

The scanner guards against a credential written by accident: a token pasted into a draft, an
environment dump captured into a log, a key file attached by mistake. It does not stand against an
agent working to get a credential past it; a credential in a shape no rule knows, or one the agent
encodes, passes. Every caller fails closed where it cannot read what it would scan (standard input,
a file not on disk, input it cannot decode), and leaves these accepted limits:

- **A staged file built from patches.** The gate scans each `mcp__sdd__stage_attachment` call's
  text, a patch's `new` text among it, not the file the patches leave behind: that file is the sdd
  engine's staged state, which the gate does not hold. A credential split across patches passes. A
  staged file's `path` is read against the session's working directory, which in every supported
  setup is the project root the sdd MCP server runs in, so it names the file the engine reads.
- **The last instant before sdd reads an attached file.** `sdd-capture` scans every attached file
  right before it runs `sdd new`, and sdd then reads the file itself: a file changed in between is
  never scanned. The gate's earlier scan of a plain `sdd new` has the same window, wider.
- **What a launcher or an expansion would change.** The gate does not resolve what the shell would
  rewrite or a launcher would run elsewhere: it refuses it (see [Credentials](#credentials)). A path
  written so that sdd reads one file and the gate another would need the gate to follow the shell,
  which it does not try to do; refusing those spellings closes that class instead.
- **A pattern that cuts a tool's name.** A command word whose pattern stands for part of a tool's
  name, with no word of the call naming `sdd`, `new` or `capture` (`sdd-* d.md`), is not seen as a
  write. Refusing every patterned command word would close it, at the cost of refusing ordinary
  commands such as `./scripts/*.sh`.
