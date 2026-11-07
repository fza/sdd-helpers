package safefs_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/safefs"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func replaceAt(root string, target string) error {
	held, name, err := safefs.Parent(root, target)
	if err != nil {
		return err
	}

	defer held.Close()

	return held.ReplaceWith(name, "", 0o600, func(file *os.File) error {
		_, err := file.WriteString("written")

		return err
	})
}

func TestAReplacementNeverFollowsALinkPlantedInsideTheRoot(t *testing.T) {
	plantings := []struct {
		name     string
		replaced bool
		plant    func(t *testing.T, root string, target string)
	}{
		{name: "a link at the file", replaced: true, plant: func(t *testing.T, root string, target string) {
			t.Helper()

			// A sibling in the same folder: the folder's os.Root follows a link
			// that stays inside it, and refuses one leaving it.
			test.WriteFile(t, filepath.Join(filepath.Dir(target), "spare.txt"), "untouched\n")

			if os.Symlink("spare.txt", target) != nil {
				t.Fatal("the link should be planted")
			}
		}},
		{name: "a link at the folder", plant: func(t *testing.T, root string, target string) {
			t.Helper()

			if os.Remove(filepath.Dir(target)) != nil || os.Symlink("decoy", filepath.Dir(target)) != nil {
				t.Fatal("the link should be planted")
			}
		}},
		{name: "a hard link at the file", plant: func(t *testing.T, root string, target string) {
			t.Helper()

			if os.Link(filepath.Join(root, "decoy", "keep.txt"), target) != nil {
				t.Fatal("the link should be planted")
			}
		}},
	}

	for _, planting := range plantings {
		t.Run(planting.name, func(t *testing.T) {
			root := t.TempDir()
			decoy := filepath.Join(root, "decoy")
			test.WriteFile(t, filepath.Join(decoy, "keep.txt"), "untouched\n")

			target := filepath.Join(root, "state", "target")

			err := os.Mkdir(filepath.Join(root, "state"), 0o700)
			if err != nil {
				t.Fatalf("making the folder: %v", err)
			}

			planting.plant(t, root, target)

			err = replaceAt(root, target)

			spare, _ := os.ReadFile(filepath.Join(root, "state", "spare.txt"))
			if !test.Untouched(t, decoy) || len(spare) > 0 && string(spare) != "untouched\n" {
				t.Error("a replacement should never land where a planted link points")
			}

			if err == nil && !planting.replaced {
				t.Error("a replacement through a planted folder link or onto a hard link should be refused")
			}

			if planting.replaced && err != nil {
				t.Errorf("a replacement over a planted link at the file should replace it, got %v", err)
			}
		})
	}
}

func TestAReplacementNeverLandsOutsideItsRoot(t *testing.T) {
	for _, planting := range test.Plantings() {
		t.Run(planting.Name, func(t *testing.T) {
			root := t.TempDir()
			outside := test.Outside(t)
			target := filepath.Join(root, "state", "target")
			planting.Plant(t, target, outside)

			_ = replaceAt(root, target)

			if !test.Untouched(t, outside) {
				t.Error("a replacement should never land outside its root")
			}
		})
	}
}

func TestAReplacementOutsideTheRootIsRefused(t *testing.T) {
	root := t.TempDir()
	outside := test.Outside(t)

	err := replaceAt(root, filepath.Join(root, "..", filepath.Base(outside), "keep.txt"))
	if !errors.Is(err, safefs.ErrOutside) || !test.Untouched(t, outside) {
		t.Errorf("a path outside the root should be refused, got %v", err)
	}
}
