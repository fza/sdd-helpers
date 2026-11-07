package process

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// sddConfigLimit bounds one `sdd config get`, which only reads files.
const sddConfigLimit = 5 * time.Second

// SddConfig reads sdd's configuration as sdd layers it for a directory.
type SddConfig struct{}

// SddValue is what `sdd config get <key>` prints in dir, trimmed, or nothing
// where sdd is missing, fails or knows no such key.
func (SddConfig) SddValue(ctx context.Context, dir string, key string) string {
	_, err := exec.LookPath("sdd")
	if err != nil {
		return ""
	}

	bounded, cancel := context.WithTimeout(ctx, sddConfigLimit)
	defer cancel()

	command := exec.CommandContext(bounded, "sdd", "config", "get", key)
	command.Dir = dir

	output, err := command.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
