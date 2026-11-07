// Package pieces is every sdd-helpers program with the plugin option that
// switches it on or off for the machine, and every tie between them: a feature
// of one program that rests on another and goes off with it. Every binary
// reads its own switch and its features' ties here, so the feature that goes
// off and the line that reports it never disagree.
package pieces

import (
	"fmt"

	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
)

const self = "sdd-helpers"

type Piece struct {
	Name   string
	Binary string
	Key    string
	on     func(options pluginconfig.Options) bool
}

func (piece Piece) On(options pluginconfig.Options) bool {
	return piece.on(options)
}

// Off is what a command of an off piece answers on stderr, exiting 1.
func (piece Piece) Off() string {
	return fmt.Sprintf("%s: off on this machine (plugin option %s)", piece.Binary, piece.Key)
}

var (
	GroundingGate = Piece{Name: "grounding gate", Binary: "sdd-grounding-gate", Key: "grounding_gate_enabled",
		on: func(options pluginconfig.Options) bool { return options.GroundingGateEnabled }}
	Capture = Piece{Name: "capture", Binary: "sdd-capture", Key: "capture_enabled",
		on: func(options pluginconfig.Options) bool { return options.CaptureEnabled }}
)

// Tie is a feature of a piece, In, that rests on another, On, and goes off
// while that one is.
type Tie struct {
	In      Piece
	On      Piece
	Feature string
	Why     string
}

var CaptureGrounded = Tie{In: Capture, On: GroundingGate, Feature: "sdd-capture",
	Why: "its grounding check is the gate's hook; without it a draft lands ungrounded"}

func Ties() []Tie {
	return []Tie{CaptureGrounded}
}

// Holds is whether the feature runs: the piece it rests on is on.
func (tie Tie) Holds(options pluginconfig.Options) bool {
	return tie.On.On(options)
}

// Off is what a command whose whole work is the tie's feature answers on
// stderr while the tie is broken, exiting 1.
func (tie Tie) Off() string {
	return fmt.Sprintf("%s: off while %s is off (plugin option %s)", tie.In.Binary, tie.On.Name, tie.On.Key)
}

// Line is how a broken tie reads at session start.
func (tie Tie) Line() string {
	return fmt.Sprintf("%s: %s is off, so %s is off; turn %s on to restore it", self, tie.On.Name, tie.Feature,
		tie.On.Key)
}

// Broken is every tie whose feature is off because the piece it rests on is,
// while the piece the feature belongs to is on.
func Broken(options pluginconfig.Options) []Tie {
	var broken []Tie

	for _, tie := range Ties() {
		if tie.In.On(options) && !tie.Holds(options) {
			broken = append(broken, tie)
		}
	}

	return broken
}
