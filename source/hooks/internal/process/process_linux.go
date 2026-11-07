package process

import (
	"context"
	"os"
)

var proc = Procfs{Root: os.DirFS("/proc")}

func (Table) Executables(_ context.Context, prefix string) ([]string, error) {
	return proc.Executables(prefix)
}
