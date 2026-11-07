package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// Executables is the executable path of every process of this user whose
// name starts with prefix, as the process was started from. kern.procargs2
// opens with the argument count, then that path. The kernel keeps the name
// truncated to 16 bytes.
func (Table) Executables(_ context.Context, prefix string) ([]string, error) {
	listed, err := unix.SysctlKinfoProcSlice("kern.proc.uid", os.Getuid())
	if err != nil {
		return nil, fmt.Errorf("listing this user's processes: %w", err)
	}

	var paths []string

	for _, info := range listed {
		name := unix.ByteSliceToString(info.Proc.P_comm[:])
		if !strings.HasPrefix(name, prefix) || Exiting(info.Proc.P_stat, info.Proc.P_flag) {
			continue
		}

		raw, err := unix.SysctlRaw("kern.procargs2", int(info.Proc.P_pid))
		if err != nil || len(raw) <= 4 {
			continue
		}

		path, _, _ := bytes.Cut(raw[4:], []byte{0})
		paths = append(paths, string(path))
	}

	return paths, nil
}
