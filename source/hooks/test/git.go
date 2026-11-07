package test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fza/sdd-helpers/source/hooks/internal/process"
)

// Git answers the questions sdd-capture asks git before it removes a draft:
// the checkout's top level, and whether git tracks a path.
type Git struct {
	Top string
	// OutsideWorkTree answers as git does in a directory that is no work tree.
	OutsideWorkTree bool
	// Tracked lists the paths `ls-files -- <path>` names as tracked.
	// ListOutsideRepository answers `ls-files` as git does outside a
	// repository, ListFails as git failing any other way.
	Tracked               []string
	ListOutsideRepository bool
	ListFails             bool
	// Repos lists work trees by their top level. Set, it answers
	// `rev-parse --show-toplevel` and `ls-files` by the directory asked, as
	// git does: from the deepest work tree holding it, else as outside any
	// repository.
	Repos []string
	// Before runs ahead of every call it answers, the way a file changes on
	// disk while git runs.
	Before func(args []string)
}

func (fake *Git) Output(_ context.Context, dir string, args ...string) (string, error) {
	if fake.Before != nil {
		fake.Before(args)
	}

	if fake.Repos != nil {
		answer, answered, err := fake.byDirectory(dir, args)
		if answered {
			return answer, err
		}
	}

	if slices.Equal(args, []string{"rev-parse", "--show-toplevel"}) {
		if fake.OutsideWorkTree {
			return "", errors.New("fatal: not a git repository")
		}

		return fake.Top + "\n", nil
	}

	if len(args) == 3 && args[0] == "ls-files" && args[1] == "--" {
		switch {
		case fake.ListOutsideRepository:
			return "", fmt.Errorf("running git %v: %w", args, process.ErrNotARepository)
		case fake.ListFails:
			return "", errors.New("fatal: unable to read index file")
		case slices.Contains(fake.Tracked, args[2]):
			return args[2] + "\n", nil
		}

		return "", nil
	}

	return "", errors.New("unexpected git call")
}

func (fake *Git) byDirectory(dir string, args []string) (string, bool, error) {
	top := ""

	for _, each := range fake.Repos {
		holds := dir == each || strings.HasPrefix(dir, each+string(filepath.Separator))
		if holds && len(each) > len(top) {
			top = each
		}
	}

	outside := fmt.Errorf("running git %v in %s: %w", args, dir, process.ErrNotARepository)

	switch {
	case slices.Equal(args, []string{"rev-parse", "--show-toplevel"}) && top == "":
		return "", true, outside
	case slices.Equal(args, []string{"rev-parse", "--show-toplevel"}):
		return top + "\n", true, nil
	case len(args) == 3 && args[0] == "ls-files" && top == "":
		return "", true, outside
	case len(args) == 3 && args[0] == "ls-files" && !strings.HasPrefix(args[2], top+string(filepath.Separator)):
		return "", true, fmt.Errorf("fatal: %s: '%s' is outside repository at '%s'", args[2], args[2], top)
	}

	return "", false, nil
}
