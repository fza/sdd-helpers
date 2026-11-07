// Package alternatives is the table of time limits agents type in place of a
// missing `timeout`, as on macOS: gtimeout, uu-timeout and perl's alarm form.
// shellwalk reads each as a launcher, so a graph read or a capture behind one
// counts as the command it runs.
//
// The table lives in code rather than in a file an agent may write: a wrong
// launcher declaration, such as `rm` declared a launcher, would have the walk
// judge the wrong word.
package alternatives

import (
	"regexp"
	"slices"
	"strings"
)

// SlotOptions / SlotDuration / SlotCommand are the named slots of an
// alternative's template, filled from the parsed call: the launcher's
// options, its duration, and the command it runs with every word after it.
// SlotDuration inside fixed text takes the duration in whole seconds.
const (
	SlotOptions  = "{opts...}"
	SlotDuration = "{dur}"
	SlotCommand  = "{cmd...}"
)

// Alternative is one program standing in for the entry's command: Program,
// then Template filled from the call.
type Alternative struct {
	Program  string
	Template []string
}

// Entry is a command whose alternatives launch the command after them:
// Template's SlotCommand marks where it starts, and the walk reads every
// alternative as a launcher. ValueOptions are the launcher's options taking
// the next word as their value.
type Entry struct {
	Name         string
	Alternatives []Alternative
	ValueOptions []string
}

// Arg is a word of a call: its text, where Literal says the shell hands it
// on as fixed text, and OneWord whether the shell certainly hands it on as
// exactly one word, never splitting it and never dropping it.
type Arg struct {
	Text    string
	Literal bool
	OneWord bool
}

var timeoutTemplate = []string{SlotOptions, SlotDuration, SlotCommand}

var Table = []Entry{
	{
		Name:         "timeout",
		ValueOptions: []string{"-s", "--signal", "-k", "--kill-after"},
		Alternatives: []Alternative{
			{Program: "timeout", Template: timeoutTemplate},
			{Program: "gtimeout", Template: timeoutTemplate},
			{Program: "uu-timeout", Template: timeoutTemplate},
			{Program: "perl", Template: []string{"-e", "alarm " + SlotDuration + "; exec @ARGV", SlotCommand}},
		},
	},
}

// Launcher is one alternative that launches a command. Prefix is the fixed
// words its template opens with before SlotCommand, nil where it opens with
// SlotOptions and so reads as a launcher by name alone.
type Launcher struct {
	Program      string
	Prefix       []string
	ValueOptions []string
}

func Launchers() []Launcher {
	var launchers []Launcher

	for _, entry := range Table {
		for _, alternative := range entry.Alternatives {
			launcher := Launcher{Program: alternative.Program, ValueOptions: entry.ValueOptions}

			at := slices.Index(alternative.Template, SlotCommand)
			if at > 0 && alternative.Template[0] != SlotOptions {
				launcher.Prefix = alternative.Template[:at]
			}

			launchers = append(launchers, launcher)
		}
	}

	return launchers
}

// perlShellCharacters are the characters that make perl's `exec` of a single
// word hand it to /bin/sh rather than run it as a program, whitespace among
// them: one such word is shell code, not a command.
const perlShellCharacters = "$&*(){}[]'\";\\|?<>~`\n \t\r\f\v"

// PrefixLaunch is the index of the word a call of a prefixed launcher runs,
// its program at args[0]: the call must spell the prefix exactly, a duration
// as whole seconds. The word after the prefix must not start with `-`, which
// perl would read as one more option, code with `-e` among them, and must be
// fixed text holding nothing perl's `exec` hands to the shell, whatever
// follows it: a word after it that expands to nothing (`$empty`, `"$@"`, a
// glob matching nothing) leaves it alone at run time, and `exec` of one word
// runs it through /bin/sh. Every word after it must stay exactly one word.
// It is -1 for any other call, which stays inline code.
func PrefixLaunch(program string, args []Arg) int {
	for _, launcher := range Launchers() {
		if launcher.Program != program || launcher.Prefix == nil || len(args) <= len(launcher.Prefix)+1 {
			continue
		}

		matched := true

		for index, fixed := range launcher.Prefix {
			arg := args[index+1]
			if !arg.Literal || !prefixWord(fixed).MatchString(arg.Text) {
				matched = false

				break
			}
		}

		at := len(launcher.Prefix) + 1
		if matched && runsAsProgram(args[at:]) {
			return at
		}
	}

	return -1
}

func runsAsProgram(command []Arg) bool {
	program := command[0]
	if !program.Literal || program.Text == "" || strings.HasPrefix(program.Text, "-") ||
		strings.ContainsAny(program.Text, perlShellCharacters) {
		return false
	}

	for _, arg := range command[1:] {
		if !arg.OneWord {
			return false
		}
	}

	return true
}

func prefixWord(fixed string) *regexp.Regexp {
	pattern := strings.ReplaceAll(regexp.QuoteMeta(fixed), regexp.QuoteMeta(SlotDuration), `[0-9]+`)

	return regexp.MustCompile("^" + pattern + "$")
}
