// Package process runs commands (a program in a process group of its own,
// git, sdd's configuration reads), starts a process that outlives its caller,
// and reads the process table through the kernel: sysctl on darwin, /proc on
// Linux. A `ps` spawn costs milliseconds, and its text follows the caller's
// locale.
package process

// Table is the machine's process table.
type Table struct{}

// darwin's process states and flags, from <sys/proc.h>: a zombie, and a
// process inside exit (`ps` shows it as E).
const (
	darwinZombie  = 5
	darwinExiting = 0x00002000
)

// Exiting is whether a darwin process with stat and flags is a zombie or on
// its way out. Such a husk keeps its pid and name while its parent has not
// reaped it, though it runs nothing any more.
func Exiting(stat int8, flags int32) bool {
	return stat == darwinZombie || flags&darwinExiting != 0
}
