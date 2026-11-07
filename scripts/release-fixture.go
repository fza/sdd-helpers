// Command release-fixture writes the fixture of one sdd-helpers release under
// source/hooks/test/fixtures/releases/<version>/: it downloads the release's
// binaries for this machine from GitHub, checks them against the release's
// checksums.txt, drives sdd-grounding-gate through a scripted session in a
// temporary checkout carrying a graph, and copies what the gate kept there
// with dump.json, every gate call that answers to that state and what the
// release answered.
//
// Run it from the repository root after each release, with that release's
// version: go run scripts/release-fixture.go v1.0.0. The network is reached
// here and never in a test; the replay test (source/hooks/internal/grounding)
// answers every committed fixture's calls with the current code.
//
// Every path the release wrote or printed that names the temporary folder is
// written with @WORK@ in its place, and the test puts its own folder back.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	repository = "fza/sdd-helpers"
	gate       = "sdd-grounding-gate"
	session    = "a1a1a1a1-0000-4000-8000-000000000001"
	workMark   = "@WORK@"
	binaryMax  = 256 << 20
	draft      = ".tmp/drafts/gate-notes.md"
)

var (
	errUsage    = errors.New("usage: go run scripts/release-fixture.go v<major>.<minor>.<patch>")
	errChecksum = errors.New("the archive does not match the release's checksums.txt")
	versionForm = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
)

// Call is one gate call of the dump: its mode, the payload it was handed and
// what the release answered.
type Call struct {
	Args   []string `json:"args"`
	Stdin  string   `json:"stdin"`
	Code   int      `json:"code"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
}

// Dump is dump.json: the release, the moment its calls ran, and the calls.
type Dump struct {
	Release string    `json:"release"`
	At      time.Time `json:"at"`
	Calls   []Call    `json:"calls"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err := run(ctx, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "release-fixture:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 || !versionForm.MatchString(args[0]) {
		return errUsage
	}

	release := args[0]

	target, err := filepath.Abs(filepath.Join("source", "hooks", "test", "fixtures", "releases", release))
	if err != nil {
		return fmt.Errorf("locating the fixture folder: %w", err)
	}

	answer, err := os.ReadFile(filepath.Join("source", "hooks", "test", "fixtures", "collapse", "search.json"))
	if err != nil {
		return fmt.Errorf("run it from the repository root: %w", err)
	}

	work, err := os.MkdirTemp("", "release-fixture-")
	if err != nil {
		return fmt.Errorf("creating the work folder: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	work, err = filepath.EvalSymlinks(work)
	if err != nil {
		return fmt.Errorf("resolving the work folder: %w", err)
	}

	bin := filepath.Join(work, "bin")

	err = fetch(ctx, release, bin)
	if err != nil {
		return err
	}

	scripted := &scripted{work: work, bin: bin, repo: filepath.Join(work, "repo"), answer: answer}

	err = scripted.prepare()
	if err != nil {
		return err
	}

	err = scripted.play(ctx)
	if err != nil {
		return err
	}

	err = copyTree(scripted.repo, filepath.Join(work, "snapshot"), "")
	if err != nil {
		return err
	}

	dump, err := scripted.dump(ctx, release)
	if err != nil {
		return err
	}

	return scripted.write(target, dump)
}

// fetch downloads the release's archive for this machine and checksums.txt,
// checks the one against the other and unpacks the binaries into bin.
func fetch(ctx context.Context, release string, bin string) error {
	version := strings.TrimPrefix(release, "v")
	archive := fmt.Sprintf("sdd-helpers_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s/", repository, release)

	sums, err := download(ctx, base+"checksums.txt")
	if err != nil {
		return err
	}

	packed, err := download(ctx, base+archive)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(packed)
	if !slices.Contains(strings.Split(string(sums), "\n"), hex.EncodeToString(sum[:])+"  "+archive) {
		return fmt.Errorf("%w: %s", errChecksum, archive)
	}

	return unpack(packed, bin)
}

func download(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("asking for %s: %w", url, err)
	}

	response, err := (&http.Client{Timeout: 5 * time.Minute}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, response.Status)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, binaryMax))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", url, err)
	}

	return data, nil
}

// unpack writes every regular file of the archive whose name is a bare sdd-
// binary into bin, nothing else.
func unpack(packed []byte, bin string) error {
	unzipped, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return fmt.Errorf("opening the archive: %w", err)
	}

	err = os.MkdirAll(bin, 0o700)
	if err != nil {
		return fmt.Errorf("creating %s: %w", bin, err)
	}

	archive := tar.NewReader(unzipped)

	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("reading the archive: %w", err)
		}

		name := header.Name
		if strings.ContainsAny(name, `/\`) || !strings.HasPrefix(name, "sdd-") || header.Typeflag != tar.TypeReg {
			continue
		}

		data, err := io.ReadAll(io.LimitReader(archive, binaryMax))
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		err = os.WriteFile(filepath.Join(bin, name), data, 0o700)
		if err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
}

// scripted is the session the release is driven through: its binaries, a
// checkout carrying a graph, and the sdd answer it collapses.
type scripted struct {
	work   string
	bin    string
	repo   string
	answer []byte
}

// prepare lays out the checkout, and an sdd standing in for the real one that
// answers `config get sessions.token_saving` with off, so collapsing runs.
func (scripted *scripted) prepare() error {
	files := map[string]string{
		"repo/.sdd/graph/2026/11/20261101-090000-d-tac-g4t.md": "---\ntype: decision\nlayer: tactical\n---\nThe gate holds a capture to its reads.\n",
		"repo/.sdd/graph/2026/11/20261102-090000-s-tac-k2p.md": "---\ntype: signal\nlayer: tactical\n---\nA capture went through ungrounded.\n",
		"repo/" + draft: "---\ntype: signal\nlayer: tactical\nconfidence: high\ntopics: [hooks]\n---\n" +
			"A draft saved while every read stood.\n",
		"sdd/sdd":            "#!/bin/sh\n[ \"$*\" = 'config get sessions.token_saving' ] && echo off && exit 0\nexit 1\n",
		"home/.claude/.keep": "",
	}

	for name, content := range files {
		path := filepath.Join(scripted.work, name)

		err := os.MkdirAll(filepath.Dir(path), 0o700)
		if err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
		}

		err = os.WriteFile(path, []byte(content), 0o700)
		if err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

// env is the environment the gate runs in: the scratch home, the stand-in sdd
// ahead of the PATH, and nothing of the environment the script runs in but
// PATH.
func (scripted *scripted) env() []string {
	return []string{
		"PATH=" + filepath.Join(scripted.work, "sdd") + string(filepath.ListSeparator) + os.Getenv("PATH"),
		"HOME=" + filepath.Join(scripted.work, "home"),
		"XDG_CACHE_HOME=" + filepath.Join(scripted.work, "cache"),
		"TMPDIR=" + filepath.Join(scripted.work, "tmp"),
		"TZ=UTC",
		"LANG=C",
		"CLAUDE_PROJECT_DIR=" + scripted.repo,
	}
}

// call runs the gate in mode with payload on its standard input.
func (scripted *scripted) call(ctx context.Context, mode string, payload map[string]any) (Call, error) {
	stdin, err := json.Marshal(payload)
	if err != nil {
		return Call{}, fmt.Errorf("encoding the payload: %w", err)
	}

	err = os.MkdirAll(filepath.Join(scripted.work, "tmp"), 0o700)
	if err != nil {
		return Call{}, fmt.Errorf("creating the scratch folder: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	command := exec.CommandContext(runCtx, filepath.Join(scripted.bin, gate), mode)
	command.Dir = scripted.repo
	command.Env = scripted.env()
	command.Stdin = bytes.NewReader(stdin)

	var out, errOut bytes.Buffer
	command.Stdout, command.Stderr = &out, &errOut

	err = command.Run()

	var exited *exec.ExitError
	if err != nil && !errors.As(err, &exited) {
		return Call{}, fmt.Errorf("%s %s: %w", gate, mode, err)
	}

	return Call{
		Args: []string{mode}, Stdin: string(stdin), Code: command.ProcessState.ExitCode(),
		Stdout: out.String(), Stderr: errOut.String(),
	}, nil
}

func bash(command string) map[string]any {
	return map[string]any{"session_id": session, "tool_name": "Bash", "tool_input": map[string]any{"command": command}}
}

func (scripted *scripted) search() map[string]any {
	return map[string]any{
		"session_id": session, "tool_name": "mcp__sdd__search", "tool_input": map[string]any{"query": "gate"},
		"tool_response": json.RawMessage(scripted.answer),
	}
}

// play runs the session whose state the fixture keeps: a start, the three
// reads, a draft saved while they stood, and an sdd answer the agent was
// shown.
func (scripted *scripted) play(ctx context.Context) error {
	steps := []struct {
		mode    string
		payload map[string]any
	}{
		{"start", map[string]any{"session_id": session, "source": "startup"}},
		{"record", bash("sdd search --term 'grounding gate'")},
		{"record", bash("sdd search --query 'a capture that skipped its reads'")},
		{"record", bash(`sdd view --layout "topics:as-list"`)},
		{"record", map[string]any{
			"session_id": session, "tool_name": "Write",
			"tool_input": map[string]any{"file_path": filepath.Join(scripted.repo, draft)},
		}},
		{"collapse", scripted.search()},
	}

	for _, step := range steps {
		called, err := scripted.call(ctx, step.mode, step.payload)
		if err != nil {
			return err
		}

		if called.Code != 0 {
			return fmt.Errorf("%s %s should succeed, got %d: %s", gate, step.mode, called.Code, called.Stderr)
		}
	}

	return nil
}

// dump runs the calls whose answers rest on the state the session left, in
// order: a question with every read standing, the same sdd answer again, a
// compaction, a capture of the grounded draft, and a question once the reads
// are forgotten.
func (scripted *scripted) dump(ctx context.Context, release string) (Dump, error) {
	held := Dump{Release: release, At: time.Now().UTC().Truncate(time.Second)}

	ask := map[string]any{
		"session_id": session, "tool_name": "AskUserQuestion", "tool_input": map[string]any{"questions": []any{}},
	}

	steps := []struct {
		mode    string
		payload map[string]any
	}{
		{"gate", ask},
		{"collapse", scripted.search()},
		{"start", map[string]any{"session_id": session, "source": "compact"}},
		{"gate", bash("sdd-capture " + draft)},
		{"gate", ask},
	}

	for _, step := range steps {
		called, err := scripted.call(ctx, step.mode, step.payload)
		if err != nil {
			return Dump{}, err
		}

		called.Stdin = strings.ReplaceAll(called.Stdin, scripted.work, workMark)
		called.Stdout = strings.ReplaceAll(called.Stdout, scripted.work, workMark)
		called.Stderr = strings.ReplaceAll(called.Stderr, scripted.work, workMark)
		held.Calls = append(held.Calls, called)
	}

	return held, nil
}

// write lays the checkout as the session left it, before the dump's calls
// changed it, and dump.json into target. The dump's calls ran on the live
// state, so the checkout is copied from the snapshot play left.
func (scripted *scripted) write(target string, dump Dump) error {
	_, err := os.Stat(target)
	if err == nil {
		return fmt.Errorf("%s exists; a fixture is written once", target)
	}

	err = copyTree(filepath.Join(scripted.work, "snapshot"), filepath.Join(target, "checkout"), scripted.work)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the dump: %w", err)
	}

	return os.WriteFile(filepath.Join(target, "dump.json"), append(encoded, '\n'), 0o644)
}

// copyTree copies every regular file below from into to, each text file with
// the work folder written as workMark where work is set.
func copyTree(from string, to string, work string) error {
	return filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if work != "" && utf8.Valid(data) && !bytes.ContainsRune(data, 0) {
			data = bytes.ReplaceAll(data, []byte(work), []byte(workMark))
		}

		relative, err := filepath.Rel(from, path)
		if err != nil {
			return fmt.Errorf("placing %s: %w", path, err)
		}

		destination := filepath.Join(to, relative)

		err = os.MkdirAll(filepath.Dir(destination), 0o755)
		if err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(destination), err)
		}

		return os.WriteFile(destination, data, 0o644)
	})
}
