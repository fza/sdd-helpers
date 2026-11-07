package process

import (
	"bytes"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// Procfs reads the process table from Root, a Linux /proc.
type Procfs struct {
	Root fs.FS
}

// Executables is the executable path of every process whose name starts with
// prefix and whose `exe` link reads, every link in it resolved: a process of
// another user keeps it unreadable. The kernel keeps the name truncated to 15
// bytes.
func (procfs Procfs) Executables(prefix string) ([]string, error) {
	listed, err := fs.ReadDir(procfs.Root, ".")
	if err != nil {
		return nil, fmt.Errorf("listing the processes: %w", err)
	}

	var paths []string

	for _, entry := range listed {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		name, running := procfs.name(pid)
		if !running || !strings.HasPrefix(name, prefix) {
			continue
		}

		path, err := fs.ReadLink(procfs.Root, entry.Name()+"/exe")
		if err == nil {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// name is field 2 of /proc/<pid>/stat, false where the process is gone or
// a zombie or dead one awaiting its reaping, which runs nothing any more. The
// name sits in parentheses and may hold spaces and parentheses of its own, so
// it ends at the last closing one; the state, field 3, follows it.
func (procfs Procfs) name(pid int) (string, bool) {
	raw, err := fs.ReadFile(procfs.Root, strconv.Itoa(pid)+"/stat")
	if err != nil {
		return "", false
	}

	opening := bytes.IndexByte(raw, '(')
	closing := bytes.LastIndexByte(raw, ')')
	if opening < 0 || closing < opening {
		return "", false
	}

	fields := strings.Fields(string(raw[closing+1:]))
	if len(fields) == 0 || fields[0] == "Z" || fields[0] == "X" {
		return "", false
	}

	return string(raw[opening+1 : closing]), true
}
