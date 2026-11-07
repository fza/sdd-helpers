package safefs_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/safefs"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func rootAt(t *testing.T, root string) *safefs.Dir {
	t.Helper()

	held, err := safefs.Root(root)
	if err != nil {
		t.Fatalf("opening the root: %v", err)
	}

	t.Cleanup(held.Close)

	return held
}

func TestEntriesListEachNameAsTheFolderHoldsIt(t *testing.T) {
	root := t.TempDir()
	outside := test.Outside(t)
	test.WriteFile(t, filepath.Join(root, "file"), "x")
	test.WriteFile(t, filepath.Join(root, "folder", "inside"), "x")

	err := os.Symlink(outside, filepath.Join(root, "link"))
	if err != nil {
		t.Fatalf("planting the link: %v", err)
	}

	listed, err := rootAt(t, root).Entries()
	if err != nil {
		t.Fatalf("listing the root: %v", err)
	}

	kinds := map[string]fs.FileMode{}
	for _, entry := range listed {
		kinds[entry.Name()] = entry.Type()
	}

	if len(kinds) != 3 || kinds["file"] != 0 || kinds["folder"] != fs.ModeDir || kinds["link"] != fs.ModeSymlink {
		t.Errorf("each entry should be listed once, a link as a link, got %v", kinds)
	}
}

func TestEntriesRefuseAFolderPastTheWalkLimit(t *testing.T) {
	root := t.TempDir()

	for index := range safefs.WalkFiles + 1 {
		test.WriteFile(t, filepath.Join(root, strconv.Itoa(index)), "")
	}

	_, err := rootAt(t, root).Entries()
	if !errors.Is(err, safefs.ErrTooMany) {
		t.Errorf("a folder holding more entries than a walk reads should be refused, got %v", err)
	}
}

func TestARenameMovesTheNameAndNothingItPointsAt(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, root string, outside string)
		moved   bool
	}{
		{name: "a folder", moved: true, prepare: func(t *testing.T, root string, _ string) {
			t.Helper()
			test.WriteFile(t, filepath.Join(root, "from", "inside"), "x")
		}},
		{name: "a link to a folder outside", moved: true, prepare: func(t *testing.T, root string, outside string) {
			t.Helper()

			if os.Symlink(outside, filepath.Join(root, "from")) != nil {
				t.Fatal("the link should be planted")
			}
		}},
		{name: "a folder onto an empty folder", prepare: func(t *testing.T, root string, _ string) {
			t.Helper()
			test.WriteFile(t, filepath.Join(root, "from", "inside"), "x")

			if os.Mkdir(filepath.Join(root, "to"), 0o700) != nil {
				t.Fatal("the folder at the new name should be made")
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			outside := test.Outside(t)
			tc.prepare(t, root, outside)

			before, err := os.Lstat(filepath.Join(root, "from"))
			if err != nil {
				t.Fatalf("reading the entry: %v", err)
			}

			err = rootAt(t, root).Rename("from", "to")

			after, statErr := os.Lstat(filepath.Join(root, "to"))
			moved := err == nil && statErr == nil && os.SameFile(before, after) && !test.Exists(t, filepath.Join(root, "from"))

			if moved != tc.moved {
				t.Errorf("a rename should move the entry itself, refusing a folder at the new name, got %v", err)
			}

			if !tc.moved && !errors.Is(err, syscall.EEXIST) {
				t.Errorf("a folder at the new name should be refused as existing, got %v", err)
			}

			if !test.Untouched(t, outside) {
				t.Error("a rename should never touch what a link points at")
			}
		})
	}
}

func TestARenameOrRemovalNamesOneEntry(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	test.WriteFile(t, filepath.Join(root, "from", "inside"), "x")

	held := rootAt(t, root)

	for _, name := range []string{"", ".", "..", "from/inside", "../root"} {
		if held.Rename(name, "to") == nil || held.Rename("from", name) == nil || held.RemoveAll(name) == nil {
			t.Errorf("a name that is no single entry should be refused: %q", name)
		}
	}

	if !test.Exists(t, filepath.Join(root, "from", "inside")) {
		t.Error("a refused name should leave the folder as it was")
	}
}

func TestRemoveAllNeverFollowsALink(t *testing.T) {
	root := t.TempDir()
	outside := test.Outside(t)
	test.WriteFile(t, filepath.Join(root, "gone", "deep", "file"), "x")

	err := os.Symlink(outside, filepath.Join(root, "gone", "deep", "link"))
	if err != nil {
		t.Fatalf("planting the link: %v", err)
	}

	err = os.Symlink(outside, filepath.Join(root, "top-link"))
	if err != nil {
		t.Fatalf("planting the link: %v", err)
	}

	held := rootAt(t, root)

	for _, name := range []string{"gone", "top-link"} {
		err = held.RemoveAll(name)
		if err != nil {
			t.Fatalf("removing %s: %v", name, err)
		}
	}

	listed, err := os.ReadDir(root)
	if err != nil || len(listed) != 0 {
		t.Errorf("the folder and the link should be gone, left %v", listed)
	}

	if !test.Untouched(t, outside) {
		t.Error("a removal should never reach through a link")
	}
}
