# AGENTS.md - `source/`

Go rules, loaded when a session touches code under `source/`.

## Layout

- **One module, `source/hooks`.** Every program is a binary under `cmd/sdd-<name>/`, published by
  each release and built for development as the root [`AGENTS.md`](../AGENTS.md) sets out. The
  `sdd-` prefix is not optional: a bare name on a PATH can shadow a system tool.
- **Every program is a binary of its own.** The grounding gate refuses a call when it fails, so no
  other program's startup can fail it.
- **`.goreleaser.yaml` lists every binary the release builds.** A new `cmd/` directory is added
  there, or the plugin's launcher finds no binary for it.
- **`main.go` resolves the environment and calls the program's `Main`, nothing else.** Every
  decision lives under `internal/`, where a test can drive it.
- **No CGO. No `/pkg`.** Every non-`cmd` package lives under `internal/`, except the module's own
  `test/` package: fakes, shared helpers and fixtures, never inline in a `_test.go`.
- **Standard library first.** A dependency earns its place by removing a mechanism the programs
  would otherwise carry and test themselves. `github.com/goccy/go-yaml` reads a draft's header,
  `.sdd/config.yaml` and `.sdd/grounding.yaml`, since the standard library parses no YAML;
  `mvdan.cc/sh/v3` parses bash for the grounding gate, the only way to tell a call from text that
  names one; `golang.org/x/sys` sets a symlink's own times in the launcher tests, which `os.Chtimes`
  cannot, since it follows the link.

## Hooks

- **A hook stays silent in a project without `.sdd/`.** It reads nothing, creates nothing and exits
  0 before it touches stdin. `sdd-grounding-gate check` is the one exception: its work is the
  machine, an option tie broken or a binary on the PATH, so it speaks in every project.

## Code

- **Assign, then check. Never inline** `if err := foo(); err != nil`.
- **Wrap with context:** `fmt.Errorf("reading the payload: %w", err)`. Never log an error and then
  return it.
- **Sentinel errors are package-level `var Err...`**, compared with `errors.Is`.
- **Accept interfaces, return structs.** An interface is declared beside the code that consumes it.
- **No `panic` outside `main`**, and none there past argument parsing.
- **Names say what a thing is for**, not what it holds.
- **Context first** on anything reaching a process or a file.
- **No comment per function, variable or constant.** Package comment above the `package` clause of
  a real source file.

## Tests

- **Run `go test -race ./...` from `source/hooks` to check the code compiles.** Never build for that.
- **Unit tests cover what a function decides; functional tests drive a program's `Main`** against a
  tree in `t.TempDir()`. No test reaches the network or the developer's home, and a subprocess is
  faked behind an interface.
- **Table-driven with `t.Run`** where the cases share a shape. In-package tests go in
  `*_internal_test.go`; every other test file declares the `<pkg>_test` package.
- **Fixtures are real files under `test/fixtures/`**, embedded with `go:embed`.
- **Assertion messages describe the behaviour**, never repeat the expected value.
