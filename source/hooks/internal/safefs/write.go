package safefs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// Writers trust no path below a root: a write never opens one whole. It holds
// the descriptor of each folder, opened relative to its verified parent as
// reads do, and creates, appends and renames relative to that descriptor. A
// file is created exclusively, a replacement goes through a fresh name renamed
// over the old one, and an appended or locked file must be the regular file
// of one name its folder names.

// Root opens root, which must exist. The root and every folder above it are
// the owner's, so a link among them is followed.
func Root(root string) (*Dir, error) {
	opened, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", root, err)
	}

	return &Dir{root: opened, path: root}, nil
}

// created holds, per root, what EnsureRoot runs once it has made that root.
var created sync.Map

// EnsureRoot is Root, making the root and every missing folder above it
// first.
func EnsureRoot(root string) (*Dir, error) {
	var made func(string)

	stored, registered := created.Load(filepath.Clean(root))
	if registered {
		_, err := os.Lstat(root)
		if errors.Is(err, fs.ErrNotExist) {
			made, _ = stored.(func(string))
		}
	}

	err := os.MkdirAll(root, 0o700)
	if err != nil {
		return nil, fmt.Errorf("creating %s: %w", root, err)
	}

	if made != nil {
		made(root)
	}

	return Root(root)
}

// EnsureSub opens the folder name below held, making it first where it is
// missing.
func (held *Dir) EnsureSub(name string, perm fs.FileMode) (*Dir, error) {
	if !singleName(name) {
		return nil, fmt.Errorf("%s names no single folder", path.Join(held.path, name))
	}

	err := held.root.Mkdir(name, perm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("creating %s: %w", path.Join(held.path, name), err)
	}

	return held.Sub(name)
}

// EnsureSubPath is EnsureSub for each component of a slash path; an empty
// path opens held itself again.
func (held *Dir) EnsureSubPath(relative string, perm fs.FileMode) (*Dir, error) {
	current := held

	for _, name := range strings.Split(relative, "/") {
		if name == "" || name == "." {
			continue
		}

		next, err := current.EnsureSub(name, perm)
		if current != held {
			current.Close()
		}

		if err != nil {
			return nil, err
		}

		current = next
	}

	if current == held {
		return held.reopen()
	}

	return current, nil
}

func (held *Dir) reopen() (*Dir, error) {
	again, err := held.root.OpenRoot(".")
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", held.path, err)
	}

	return &Dir{root: again, path: held.path}, nil
}

// CreateFile opens a new file below held for writing, refusing one already
// there, a link included: O_EXCL never opens an existing name. The file gets
// perm whatever the umask.
func (held *Dir) CreateFile(name string, perm fs.FileMode) (*os.File, error) {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return nil, fmt.Errorf("%s names no single file", at)
	}

	file, err := held.root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, fmt.Errorf("creating %s: %w", at, err)
	}

	err = file.Chmod(perm)
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("setting the mode of %s: %w", at, err)
	}

	return file, nil
}

// CreateTemp creates a new file below held named by pattern, its last `*`
// replaced by a random string, as os.CreateTemp names one. It answers the file
// and its name.
func (held *Dir) CreateTemp(pattern string, perm fs.FileMode) (*os.File, string, error) {
	prefix, suffix := pattern, ""

	at := strings.LastIndex(pattern, "*")
	if at >= 0 {
		prefix, suffix = pattern[:at], pattern[at+1:]
	}

	for attempt := 0; ; attempt++ {
		random := make([]byte, 8)

		_, err := rand.Read(random)
		if err != nil {
			return nil, "", fmt.Errorf("naming a temporary file: %w", err)
		}

		name := prefix + hex.EncodeToString(random) + suffix

		file, err := held.CreateFile(name, perm)
		if err == nil {
			return file, name, nil
		}

		if !errors.Is(err, fs.ErrExist) || attempt >= 16 {
			return nil, "", err
		}
	}
}

// Replace writes name below held through a fresh name renamed over it: the
// fresh name is never one a planted link holds, and a rename replaces a link
// at name rather than writing through it.
func (held *Dir) Replace(name string, data []byte, perm fs.FileMode) error {
	return held.ReplaceWith(name, "", perm, func(file *os.File) error {
		_, err := file.Write(data)

		return err
	})
}

// ReplaceWith is Replace with the content written by fill, into a temporary
// file named by pattern ("." + name + "-*" where empty).
func (held *Dir) ReplaceWith(name string, pattern string, perm fs.FileMode, fill func(*os.File) error) error {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return fmt.Errorf("%s names no single file", at)
	}

	// A rename never writes into the file it replaces, but a second name says
	// a process outside planted it, and the owner learns of it here.
	existing, err := held.root.Lstat(name)
	if err == nil && existing.Mode().IsRegular() && hardLinked(existing) {
		return fmt.Errorf("%s %w", at, ErrHardLinked)
	}

	if pattern == "" {
		pattern = "." + name + "-*"
	}

	file, temporary, err := held.CreateTemp(pattern, perm)
	if err != nil {
		return err
	}

	err = fill(file)

	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}

	if err == nil {
		err = held.root.Rename(temporary, name)
	}

	if err != nil {
		_ = held.root.Remove(temporary)

		return fmt.Errorf("writing %s: %w", at, err)
	}

	return nil
}

// OpenChecked opens name below held with flag, never blocking on a pipe, and
// refuses the descriptor before anything is written unless it is the regular
// file of one name the folder names: os.Root follows a link inside its tree.
func (held *Dir) OpenChecked(name string, flag int, perm fs.FileMode) (*os.File, error) {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return nil, fmt.Errorf("%s names no single file", at)
	}

	file, err := held.root.OpenFile(name, flag|syscall.O_NONBLOCK, perm)

	// darwin answers an openat with O_CREAT|O_NOFOLLOW racing another creating
	// the same name with ENOENT now and then; the name exists by the next try.
	for attempt := 0; flag&os.O_CREATE != 0 && errors.Is(err, fs.ErrNotExist) && attempt < 16; attempt++ {
		file, err = held.root.OpenFile(name, flag|syscall.O_NONBLOCK, perm)
	}

	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", at, err)
	}

	opened, err := file.Stat()
	if err == nil {
		var checked fs.FileInfo

		checked, err = held.FileInfo(name)
		if err == nil && (!opened.Mode().IsRegular() || !os.SameFile(checked, opened) || hardLinked(opened)) {
			err = fmt.Errorf("%s %w", at, ErrMoved)
		}
	}

	if err != nil {
		_ = file.Close()

		return nil, err
	}

	return file, nil
}

// Rename moves the entry from to the name to, both below held. A rename
// never follows a link at either name, and os.Root refuses a folder already
// at the new name, as os.Rename does, where rename(2) would replace an empty
// one.
func (held *Dir) Rename(from string, to string) error {
	source, target := path.Join(held.path, from), path.Join(held.path, to)
	if !singleName(from) || !singleName(to) {
		return fmt.Errorf("moving %s to %s: names no single entry", source, target)
	}

	err := held.root.Rename(from, to)
	if err != nil {
		return fmt.Errorf("moving %s to %s: %w", source, target, err)
	}

	return nil
}

// RemoveAll removes name below held and everything below it, following no
// link.
func (held *Dir) RemoveAll(name string) error {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return fmt.Errorf("%s names no single entry", at)
	}

	err := held.root.RemoveAll(name)
	if err != nil {
		return fmt.Errorf("removing %s: %w", at, err)
	}

	return nil
}

// Remove removes the file or empty folder name below held.
func (held *Dir) Remove(name string) error {
	at := path.Join(held.path, name)
	if !singleName(name) {
		return fmt.Errorf("%s names no single entry", at)
	}

	err := held.root.Remove(name)
	if err != nil {
		return fmt.Errorf("removing %s: %w", at, err)
	}

	return nil
}
