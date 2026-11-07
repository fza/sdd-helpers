# The plugin: architecture

## Layout

| Path | Holds |
|---|---|
| `.claude-plugin/marketplace.json` | the marketplace, naming the plugin in `plugin/` |
| `plugin/.claude-plugin/plugin.json` | the plugin's manifest: its `version`, which names the release the launcher runs, and its `userConfig` options, every switch declaring its built-in `default` |
| `plugin/hooks/hooks.json` | every hook the plugin wires, each through `plugin/hooks/run` |
| `plugin/hooks/run` | the launcher |
| `plugin/bin/` | the commands the Bash tool finds, each handing its call to the launcher |
| `plugin/skills/` | one skill per piece |
| `plugin/README.md` | what the plugin is, and everything it runs, sends, writes and decides, for a user reading the installed plugin |
| `source/hooks/` | one Go module: under `cmd/`, a program per binary (`sdd-grounding-gate`, `sdd-capture`); everything else under `internal/`, the pieces, their switches and the tie between them in `internal/pieces` |

Claude Code loads every component it finds at a plugin's root, so nothing but the plugin's own
files goes into `plugin/`. The plugin carries no MCP server and no hooks module.

## Hooks

| Event | Binaries |
|---|---|
| SessionStart | `sdd-grounding-gate start`, `sdd-grounding-gate check`, both run with the launcher's `--session-start`, so either may fetch the release |
| PreToolUse on `AskUserQuestion`, `Write`, `Edit`, `NotebookEdit`, `Bash`, `Monitor`, `mcp__sdd__show`, `mcp__sdd__start_procedure`, `mcp__sdd__next`, `mcp__sdd__stage_attachment` | `sdd-grounding-gate gate` |
| PreToolUse on `Agent`, `Task` | `sdd-grounding-gate spawn` |
| PostToolUse on `Bash`, `mcp__sdd__search`, `mcp__sdd__view`, `mcp__sdd__start_procedure`, `mcp__sdd__next`; on `Write`, `Edit`; on `mcp__sdd__stage_attachment` | `sdd-grounding-gate record` |
| PostToolUse on `mcp__sdd__.*` | `sdd-grounding-gate collapse` |
| SessionEnd | `sdd-grounding-gate end` |

What each mode does is in the [grounding gate's architecture](../grounding-gate/architecture.md#modes).

## Writes

A process allowed to write a repository's committed files can plant a link where the next write
goes. So every file sdd-helpers creates, opens for writing, renames or makes a folder for below a
root goes through `internal/safefs`. The roots are the cache home (`$XDG_CACHE_HOME/sdd-helpers`,
`~/.cache/sdd-helpers` where unset, holding `sdd-caps.json`), a checkout's top folder (a draft's
lint body), the folder holding a graph's directory (`.sdd/tmp/capture.lock`, the grounding gate's
ledgers and shown sets) and the plugin's data directory (the release prune's status, its locks, and
the release folders it renames and removes). A root and every folder above it are the owner's and
may be links: a symlinked `~/.cache` keeps working.

Below the root no path is opened whole. Each component is lstat-checked, refused as a link, opened
relative to its parent's descriptor through `os.Root` and held to the checked folder, so a folder
swapped for a link in between is caught; `os.Root` alone follows a link that stays inside its tree.
A new file is created with `O_EXCL`; a replaced file is written under a fresh random name and renamed
over the old one, so a link at the old name is replaced, never written through; a locked file must
be the regular file of one name its folder names before a byte is written. A file with a second
name, a hard link, is refused wherever it is written. A refusal names the path.

## Where the binaries come from

The plugin holds no binaries. Every hook runs through
`plugin/hooks/run [--session-start | --command] <binary> [argument...]`, which runs the binary of
the release named by the plugin's own `version`, kept under the plugin's data directory,
`${CLAUDE_PLUGIN_DATA}`, or `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/plugins/data/sdd-helpers-sdd-helpers`
for a command the Bash tool runs, which gets neither plugin variable and finds the plugin from its
own path:

1. `SDD_HELPERS_BIN_DIR` set: the binary in that directory runs, and nothing is fetched.
2. The plugin's version is there (`<data>/<version>/<binary>`): it runs.
3. At SessionStart (`--session-start`), or for a command an agent runs from `bin/` (`--command`),
   the release archive for this machine, `sdd-helpers_<version>_<os>_<arch>.tar.gz`, and the
   release's `checksums.txt` are downloaded from
   `https://github.com/fza/sdd-helpers/releases/download/v<version>/`; the archive is unpacked only
   where its sha256 matches. A tool-call hook never downloads.
4. Otherwise the newest version already there of the plugin's own major runs. A session that cannot
   write the data directory, such as a sandboxed job, runs what an earlier session fetched. A
   release of another major never stands in: `hooks.json` calls the programs by its own major's
   modes, which another major's program may not know, answering with its usage line on every hook
   event.
5. No binary at all: `sdd-grounding-gate gate` refuses every call it guards (exit 2), naming the
   fix, rather than let work run ungrounded; a `--command` call exits 1 on stderr, since a silent
   success would read as an entry captured; every other hook exits 0. The SessionStart that failed
   to fetch says why, once.

A `plugin.json` naming no `version`, as before the first release (Claude Code then versions the
plugin by its commit SHA), names no release: the launcher fetches nothing and no other version
stands in, so the gate refuses every call, saying the plugin names no release yet, until
`SDD_HELPERS_BIN_DIR` is set or a release is cut. A release writes its version into `plugin.json`.

The archive holds `sdd-grounding-gate` and `sdd-capture`, one program per binary. It is unpacked in
the fetch's work directory and renamed into place, so a version directory is complete or absent.

## Release pruning

Each release unpacks into a folder of its own, and the launcher never removes one, so the daily
release prune (`internal/releaseprune`) deletes the folders no session may still run. A session
still running an older plugin version runs that version's programs, and without them its gate
refuses every call, so a folder goes only once no session can need it. Kept:

- the version `plugin/.claude-plugin/plugin.json` names;
- every version a running process whose name starts with `sdd-` executes a program from, read from
  the process table through the kernel (`sysctl` `kern.proc.uid` and `kern.procargs2` on darwin,
  `/proc/<pid>/stat` and `/proc/<pid>/exe` on Linux), a zombie or exiting process left out. A path
  counts against the data directory as named and with every link resolved, since darwin reports the
  path a process was started from and Linux the resolved one;
- every version a SessionStart used within 14 days. The launcher writes `<data>/<version>/used`
  before it runs a binary of that version at a SessionStart, whichever version runs: the named one,
  one just fetched, or one standing in. The mark is written by the shell's builtin `echo`, so it
  spawns no process; a folder the session cannot write, as in a sandboxed job, keeps its mark as it
  was and the binary still runs.

A run that cannot read the plugin's named version or the process table keeps every release. Only a
folder named as a version (`^[0-9]+\.[0-9]+\.[0-9]+$`) is ever removed: a link, a file, a
`.fetch-` lock or work directory, and every other name stay. A folder is renamed to
`.pruned-<version>` and then removed, so the launcher never finds half a version under its name; a
`.pruned-<version>` folder a crash left is removed first. The data directory is the root of every
rename and removal through `internal/safefs`: a link to it is followed, none below it ever is.

`sdd-grounding-gate start` and `sdd-grounding-gate check` both start the prune. Each reads
`<data>/release-prune.json` and, where no run started within the last day, stamps the start under
`<data>/release-prune.lock`, taken without waiting, then starts `sdd-grounding-gate prune` from its
own executable in a session of its own, its output discarded, so no hook waits for it; of the
SessionStart hooks of every session asking, the first alone starts one. On a day the run started
already, the check is one read of the status. The run inherits `CLAUDE_PLUGIN_DATA` and
`CLAUDE_PLUGIN_ROOT`, holds `<data>/release-prune.run.lock` so a second run exits at once, stops
after five minutes, and records its outcome in `release-prune.json`:

| Key | Holds |
|---|---|
| `started` | when the day's run was claimed |
| `finished` | when the run ended |
| `deleted` | each release folder it removed |
| `error` | why the run failed, where it did |
| `warned` | a SessionStart told the error already |
| `denied` | how much the run was not allowed to read or touch, as in a sandbox; such a run is claimed again an hour later rather than a day |

The next `start` or `check` prints a failed run's error once, as `sdd-helpers: release prune failed
at <finished>: <error>; see <data>/release-prune.json`, and sets `warned`, so a failure is told once
until a run fails anew. `start` works only in a project carrying a graph, with the gate on; `check`
runs at every session start in every project, whatever the gate's switch, so a machine that never
opens a graph project, or has the gate off, still prunes. The day's claim keeps the two to one run.

## The fetch lock

The SessionStart hooks run in parallel, so the first to fetch holds a lock, `.fetch-<version>` in
the data directory, and the rest wait for it, up to 100 seconds.

- **Owner**: the lock is a symlink whose target is the fetching process's pid, created with
  `ln -s`. `symlink(2)` creates the name and its target in one call and fails on a name that exists,
  on darwin and Linux alike, so no lock is ever visible without its owner. A lock directory with the
  pid written into it afterwards leaves a gap a `SIGKILL` can land in, and renaming a prepared
  directory into place cannot refuse an existing one, since macOS `mv` has no `-T`.
- **Work**: the fetch downloads and unpacks in `.fetch-<version>.<pid>/`, beside the lock.
- **Signals**: on `HUP`, `INT` or `TERM` the fetch removes its lock and its work and stops its
  download, which runs in the background so the trap fires at once.
- **Takeover**: a waiter finding the owner's process gone, as after `SIGKILL`, removes the lock and
  that owner's work at once and fetches itself; a `.fetch-<version>.takeover` directory keeps two
  waiters from both doing so. A lock whose owner runs is waited on, however old. A lock directory
  with the pid in its `owner` file is honoured the same way, and one with no `owner` file is taken
  over once older than five minutes.

## The release

A release is cut by starting the `release` workflow by hand (`gh workflow run release.yml`, or the
Actions tab). It runs `test.yml` first, then semantic-release (`.releaserc.yml`) reads the commits
since the last tag through the `conventionalcommits` preset: `fix:` cuts a patch, `feat:` a minor, a
`!` after the type a major, and every other type, `sdd:` included, cuts nothing. A releasable run
writes the new version into `plugin/.claude-plugin/plugin.json` with `jq`, commits it to `main` as
`chore(release): v<version> [skip ci]`, tags that commit, and creates the GitHub release as a draft
with the generated notes; GoReleaser (`goreleaser release --clean`, `.goreleaser.yaml`) builds
`sdd-capture` and `sdd-grounding-gate` for darwin and linux, amd64 and arm64, with CGO off and
`-trimpath`, adds an archive per platform and `checksums.txt` to that draft, and publishes it. The
repository's releases are immutable, so nothing can be added to a release once it is published: the
draft is what lets the archives land at all. The first release is v1.0.0. A breaking change marked
without `!` ships as a minor. Pull before pushing after a release: `main` carries the release commit.

The release notes never pass through a shell: `@semantic-release/github` creates the release
carrying them, and GoReleaser uploads the archives into it, keeping its notes. Interpolating them
into a command would let a commit subject run as shell under a token that writes to the repository.

A run that failed after pushing its tag and draft is finished from that tag: the workflow's
`publish` input names it (`v1.2.3`), GoReleaser builds there and publishes the draft, and
semantic-release and the tests do not run, since the tag was tested before its release commit,
which changes only the version.

### Release fixtures

A release can never lose a session's reads, grounded drafts or shown sets to the next one. After
each release, before the next one, `scripts/release-fixture v<version>` (running
`scripts/release-fixture.go`) downloads that release's binaries for the machine from GitHub, checks
them against the release's `checksums.txt`, drives `sdd-grounding-gate` through a scripted session
in a temporary checkout carrying a graph (reads recorded, a draft saved, an sdd answer collapsed),
and writes `source/hooks/test/fixtures/releases/<version>/`: the gate's state under
`.sdd/tmp/grounding-gate/`, and `dump.json`, each gate call and what the release answered. `@WORK@`
stands for the folder the release ran in. The network is reached in the script and never in a test.

A test replays every committed fixture against the current code and expects the same answers.
Another holds the fixtures and `plugin.json` together: with no version named no fixture may exist,
and the version `plugin.json` names must have one. So the `release` workflow, which runs the tests
first, cannot cut a release until the fixture of the one before it is committed.

## Skill budgets

A skill's `SKILL.md` loads whole into context each time it is needed, so its growth is paid on every
load. Each one has a budget in characters, `skillBudgets` in
`source/hooks/internal/plugin/skillbudget_test.go`, recorded there with the reason for its figure;
the test fails where a `SKILL.md` exceeds its budget, where a skill has none, and where a budget
names no skill. Raising one is a change of its own, stating what the added text earns. Rejected: one
budget for the whole skill directory, under which one file's growth hides behind another's cut.

## Tests

```bash
cd source/hooks && go vet ./... && go test -race ./...   # every Go suite
scripts/linux-test                                       # every Go suite on Linux, in a container
claude plugin validate .                                 # the marketplace manifest
claude plugin validate plugin                            # the plugin manifest
```

The rules for this code live in [`source/AGENTS.md`](../../source/AGENTS.md). Each suite drives its
binary's `Main` against a tree built in a temporary directory, with processes faked, never the real
ones. The plugin's suite runs the real `plugin/hooks/run` and `plugin/bin/` shim against a fake
`curl` and a temporary data directory, and checks that every hook in `plugin/hooks/hooks.json` goes
through the launcher and that the release builds every binary under `source/hooks/cmd/`.

`test.yml` runs `go vet` and `go test -race` on ubuntu and macOS for each push, and validates the
marketplace and the plugin. `scripts/linux-test` runs the same suites locally on Linux, in a
container of Apple's [`container`](https://github.com/apple/container) tool; its arguments go to
`go test` (`./...` when it gets none).
