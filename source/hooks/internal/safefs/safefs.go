// Package safefs reads and writes below a root without following a link.
//
// A process allowed to write part of the data home, or a repository's
// committed files, can plant a link where the owner's next write goes, or swap
// a checked folder for one before the write opens it. So a path below a root
// is never opened whole: each component is lstat-checked, refused as a link,
// opened relative to its parent's descriptor through os.Root, and held to the
// checked folder, so a link swapped in between is caught rather than followed.
// os.Root alone follows a link that stays inside its tree. The root and every
// folder above it are the owner's and may be links.
package safefs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
)

var (
	ErrLink       = errors.New("is a link")
	ErrNotRegular = errors.New("is no regular file")
	ErrTooLarge   = errors.New("is over the read limit")
	ErrTooMany    = errors.New("holds more files than a walk reads")
	ErrMoved      = errors.New("changed while it was opened")
	ErrHardLinked = errors.New("has a second name, a hard link; register a copy of it")
	ErrOutside    = errors.New("lies outside its root")
)

// A walk reads at most WalkDepth folders deep and WalkFiles files.
const (
	WalkDepth = 6
	WalkFiles = 2000
)

// Dir is a directory opened by descriptor.
type Dir struct {
	root *os.Root
	path string
}

// A link's own lstat is never a folder or a regular file, so the type checks
// refuse it; these name it for what it is.
func notADirectory(at string, checked fs.FileInfo) error {
	if checked.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s %w", at, ErrLink)
	}

	return fmt.Errorf("%s is no directory", at)
}

func notARegularFile(at string, checked fs.FileInfo) error {
	if checked.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s %w", at, ErrLink)
	}

	return fmt.Errorf("%s %w", at, ErrNotRegular)
}

func (held *Dir) Close() {
	if held != nil {
		_ = held.root.Close()
	}
}

func singleName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

// Sub opens one named directory below held.
func (held *Dir) Sub(name string) (*Dir, error) {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return nil, fmt.Errorf("%s names no single entry", at)
	}

	checked, err := held.root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", at, err)
	}

	if !checked.IsDir() {
		return nil, notADirectory(at, checked)
	}

	child, err := held.root.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", at, err)
	}

	opened, err := child.Stat(".")
	if err != nil || !os.SameFile(checked, opened) {
		_ = child.Close()

		return nil, fmt.Errorf("%s %w", at, ErrMoved)
	}

	return &Dir{root: child, path: at}, nil
}

// SubPath opens a slash-separated path below held, one component at a time.
func (held *Dir) SubPath(relative string) (*Dir, error) {
	current := held

	for _, name := range strings.Split(relative, "/") {
		next, err := current.Sub(name)
		if current != held {
			current.Close()
		}

		if err != nil {
			return nil, err
		}

		current = next
	}

	if current == held {
		return nil, fmt.Errorf("%s names no directory below %s", relative, held.path)
	}

	return current, nil
}

// FileInfo is a regular file below held, checked without opening it.
func (held *Dir) FileInfo(name string) (fs.FileInfo, error) {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return nil, fmt.Errorf("%s names no single entry", at)
	}

	checked, err := held.root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", at, err)
	}

	if !checked.Mode().IsRegular() {
		return nil, notARegularFile(at, checked)
	}

	// A second name can be an outside file linked in: reading it would hand
	// that file out, and writing it would write there.
	if hardLinked(checked) {
		return nil, fmt.Errorf("%s %w", at, ErrHardLinked)
	}

	return checked, nil
}

// Entries lists held's own entries, at most WalkFiles of them, each as the
// folder names it: a link is listed as a link, never followed.
func (held *Dir) Entries() ([]fs.DirEntry, error) {
	opened, err := held.root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", held.path, err)
	}

	defer func() { _ = opened.Close() }()

	listed, err := opened.ReadDir(WalkFiles + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("listing %s: %w", held.path, err)
	}

	if len(listed) > WalkFiles {
		return nil, fmt.Errorf("%s %w", held.path, ErrTooMany)
	}

	return listed, nil
}

// Walked is one regular file a walk found, by its slash path below the walk's
// start.
type Walked struct {
	Name string
	Data []byte
}

// hardLinked says whether a file has a second name.
func hardLinked(info fs.FileInfo) bool {
	stat, readable := info.Sys().(*syscall.Stat_t)

	return readable && stat.Nlink > 1
}
