package safefs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Each function here takes a root and a full path below it, and walks from the
// root to the path's folder one component at a time.

func relative(root string, target string) (string, error) {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s %w %s", target, ErrOutside, root)
	}

	return filepath.ToSlash(rel), nil
}

// At opens the existing folder dir below root.
func At(root string, dir string) (*Dir, error) {
	rel, err := relative(root, dir)
	if err != nil {
		return nil, err
	}

	top, err := Root(root)
	if err != nil {
		return nil, err
	}

	if rel == "." {
		return top, nil
	}

	defer top.Close()

	return top.SubPath(rel)
}

// EnsureAt opens the folder dir below root, making the root and every missing
// folder down to dir first, each folder below the root with perm.
func EnsureAt(root string, dir string, perm fs.FileMode) (*Dir, error) {
	rel, err := relative(root, dir)
	if err != nil {
		return nil, err
	}

	top, err := EnsureRoot(root)
	if err != nil {
		return nil, err
	}

	if rel == "." {
		return top, nil
	}

	defer top.Close()

	return top.EnsureSubPath(rel, perm)
}

// MkdirAll is EnsureAt, closing the folder again.
func MkdirAll(root string, dir string, perm fs.FileMode) error {
	held, err := EnsureAt(root, dir, perm)
	if err != nil {
		return err
	}

	held.Close()

	return nil
}

// Parent opens the existing folder of target below root, and answers target's
// own name in it.
func Parent(root string, target string) (*Dir, string, error) {
	rel, err := relative(root, target)
	if err != nil {
		return nil, "", err
	}

	if rel == "." {
		return nil, "", fmt.Errorf("%s is its root and no entry below it", target)
	}

	held, err := At(root, filepath.Join(root, filepath.FromSlash(path.Dir(rel))))
	if err != nil {
		return nil, "", err
	}

	return held, path.Base(rel), nil
}

// WriteFile replaces target below root with data, through a fresh name renamed
// over it; its folder must exist.
func WriteFile(root string, target string, data []byte, perm fs.FileMode) error {
	held, name, err := Parent(root, target)
	if err != nil {
		return err
	}

	defer held.Close()

	return held.Replace(name, data, perm)
}

// OpenChecked opens target below root with flag, refused unless it is the
// regular file of one name its folder names; its folder must exist.
func OpenChecked(root string, target string, flag int, perm fs.FileMode) (*os.File, error) {
	held, name, err := Parent(root, target)
	if err != nil {
		return nil, err
	}

	defer held.Close()

	return held.OpenChecked(name, flag, perm)
}

func Lock(root string, target string, perm fs.FileMode) (*os.File, error) {
	return OpenChecked(root, target, os.O_CREATE|os.O_RDWR, perm)
}
