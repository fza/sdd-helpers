package pieces

import (
	"context"
	"fmt"
)

// ForkRelease is the first release of the fza/sdd fork carrying the work of
// every piece: its capture procedure gates graph writes on the reads made
// through `gather`, scans them for credentials and collapses repeated output.
const ForkRelease = "v0.18.0-alpha.3.fza.43"

const uninstall = "`claude plugin uninstall sdd-helpers@sdd-helpers`"

// DeprecatedLine is the SessionStart line wherever the pieces still work.
const DeprecatedLine = "deprecated: sdd from the fza/sdd fork " + ForkRelease + " does its work; switch to it " +
	"(`brew install --cask fza/tap/sdd`) or keep upstream sdd, then " + uninstall

// AsideLine is the SessionStart line in a project where sdd gates graph
// writes itself, mode being the verify.mode sdd reports.
func AsideLine(mode string) string {
	return fmt.Sprintf("deprecated and standing aside: sdd's verify.mode is `%s`, so sdd gates graph writes itself "+
		"and the grounding gate and sdd-capture are off; %s removes it", mode, uninstall)
}

// CaptureAside is what sdd-capture answers on stderr, exiting 1, where sdd
// gates graph writes itself.
func CaptureAside(mode string) string {
	return fmt.Sprintf("sdd-capture: off while sdd gates graph writes itself (verify.mode `%s`); "+
		"capture through sdd's own capture procedure instead", mode)
}

// SddGates is whether the sdd run from dir gates graph writes itself, with
// the verify.mode it reports. The fork gates them under every mode but off;
// upstream sdd and fork releases before ForkRelease know no such key and
// print nothing. A version string cannot tell: earlier fork releases carry
// the fork's suffix without the gate.
func SddGates(ctx context.Context, values SddValues, dir string) (string, bool) {
	if values == nil {
		return "", false
	}

	mode := values.SddValue(ctx, dir, "verify.mode")

	return mode, mode != "" && mode != "off"
}
