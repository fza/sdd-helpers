package process_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/fza/sdd-helpers/source/hooks/internal/process"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func TestADarwinProcessOnItsWayOutIsGone(t *testing.T) {
	cases := []struct {
		name    string
		stat    int8
		flags   int32
		exiting bool
	}{
		{name: "running", stat: 2},
		{name: "sleeping with other flags", stat: 3, flags: 0x4004},
		{name: "a zombie", stat: 5, exiting: true},
		{name: "inside exit", stat: 2, flags: 0x00002000, exiting: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if process.Exiting(tc.stat, tc.flags) != tc.exiting {
				t.Error("only a zombie or a process inside exit should read as on its way out")
			}
		})
	}
}

func TestTheTableNamesTheExecutableOfEachMatchingProcess(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("finding this test's executable: %v", err)
	}

	// The kernel keeps a process's name truncated; the test binary's own name
	// is short enough to stand whole.
	paths, err := process.Table{}.Executables(context.Background(), filepath.Base(self))
	if err != nil {
		t.Fatalf("listing executables: %v", err)
	}

	wanted, _ := filepath.EvalSymlinks(self)

	found := slices.ContainsFunc(paths, func(path string) bool {
		resolved, _ := filepath.EvalSymlinks(path)

		return resolved == wanted
	})
	if !found {
		t.Errorf("the executable of this very process should be listed, got %v", paths)
	}

	none, err := process.Table{}.Executables(context.Background(), "no-such-process-name-")
	if err != nil || len(none) != 0 {
		t.Errorf("a prefix no process carries should list nothing, got %v %v", none, err)
	}
}

func TestProcfsNamesTheExecutableOfEachMatchingProcess(t *testing.T) {
	root := t.TempDir()
	test.FixtureTree(t, "process/sdd", root)

	links := map[string]string{
		"5100": "/opt/old/sdd-grounding-gate",
		"5300": "/bin/bash",
		"5400": "/opt/zombie/sdd-capture",
		"5500": "/opt/parenthesised/sdd-x",
		"5600": "/opt/zombie-parenthesised/sdd-z",
	}

	for pid, target := range links {
		err := os.Symlink(target, filepath.Join(root, pid, "exe"))
		if err != nil {
			t.Fatalf("linking the executable of %s: %v", pid, err)
		}
	}

	paths, err := process.Procfs{Root: os.DirFS(root)}.Executables("sdd-")
	if err != nil {
		t.Fatalf("listing executables: %v", err)
	}

	slices.Sort(paths)

	want := []string{"/opt/old/sdd-grounding-gate", "/opt/parenthesised/sdd-x"}
	if !slices.Equal(paths, want) {
		t.Errorf("only a running matching process whose executable reads should be listed, got %v", paths)
	}
}

func TestProcfsWithoutATableFails(t *testing.T) {
	_, err := process.Procfs{Root: os.DirFS(filepath.Join(t.TempDir(), "none"))}.Executables("sdd-")
	if err == nil {
		t.Error("a process table that cannot be listed should fail the listing")
	}
}

func TestADetachedProcessRunsInASessionOfItsOwn(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "pid")

	script := `echo $$ > "$1.tmp" && mv "$1.tmp" "$1"; echo noise; sleep 5`

	err := process.Detached{}.Start(context.Background(), "/bin/sh", "-c", script, "sh", marker)
	if err != nil {
		t.Fatalf("starting the process: %v", err)
	}

	var written []byte

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		written, err = os.ReadFile(marker)
		if err == nil {
			break
		}
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(written)))
	if err != nil {
		t.Fatalf("the detached process should run and write its pid: %v", err)
	}

	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGTERM) })

	session, err := unix.Getsid(pid)
	if err != nil || session != pid {
		t.Errorf("the detached process should lead a session of its own, got %d %v", session, err)
	}

	err = process.Detached{}.Start(context.Background(), filepath.Join(dir, "missing"))
	if err == nil {
		t.Error("a program that cannot start should fail the start")
	}
}
