package shellwalk

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ShellCode is the code a shell given args runs from its -c option, and
// whether the walk places it for certain. Certain, it is the words the shell
// runs, or none. Uncertain, it is every word from the first one the walk
// cannot place on, any of which the shell may run.
//
// fish runs the value of every -c and -C, its init command, and of the long
// --command and --init-command, as readShellArguments reads them. Every other
// shell is read twice, and every word either reading takes for code is
// handed on: by its option tables (readShellArguments), and by placedCode,
// which knows where the place of the code is lost. Only placedCode decides
// whether it is certain.
func ShellCode(name string, args []*syntax.Word) ([]*syntax.Word, bool) {
	if name == fishShell {
		return readFishArguments(args).runCode(), true
	}

	code, certain := placedCode(name, args)

	for _, word := range readShellArguments(name, args).code {
		if slices.Contains(code, word) {
			continue
		}

		at := slices.Index(args, word)

		switch {
		case !certain && len(code) > 0 && at < slices.Index(args, code[0]):
			code = args[at:]
		case certain:
			code = append(slices.Clone(code), word)
			slices.SortFunc(code, func(first *syntax.Word, second *syntax.Word) int {
				return slices.Index(args, first) - slices.Index(args, second)
			})
		}
	}

	return code, certain
}

// placedCode is the code a shell other than fish runs from its -c option,
// and whether its place is certain. The shell takes its first operand once
// its options end, so options may stand between -c and the code
// (`bash -c -e code`): -o and +o take a value, bash's -O and +O too, `--` and
// `-` end the options, and a long option ends them only in error. A word with
// an expansion or a glob, a long option no shell list knows, and a group whose
// value the shells read apart (zsh's -oe takes e as its value, bash's the next
// word) leave the code's place unknown.
func placedCode(name string, args []*syntax.Word) ([]*syntax.Word, bool) {
	if !slices.ContainsFunc(args, carriesCommand) {
		return nil, true
	}

	command := false

	for index := 0; index < len(args); index++ {
		value, isLiteral := Literal(args[index])
		short := len(value) > 1 && (value[0] == '-' || value[0] == '+') && value[1] != '-'

		switch {
		case (!isLiteral || Patterned(args[index])) && command && index == len(args)-1:
			// The last word is the code, or an option leaving the shell none.
			return args[index:], true
		case !isLiteral || Patterned(args[index]):
			return args[index:], false
		case value == "--" || value == "-":
			if command && index+1 < len(args) {
				return args[index+1 : index+2], true
			}

			return nil, true
		case slices.Contains(longFlags, value):
		case slices.Contains(longValueOptions, value):
			index++
		case strings.HasPrefix(value, "--"):
			return args[index:], false
		case short:
			command = command || value[0] == '-' && strings.Contains(value, "c")

			values, known := groupValues(name, value[1:])
			if !known {
				return args[index:], false
			}

			for range values {
				index++
				if index < len(args) && !OneKnownWord(args[index]) {
					return args[index:], false
				}
			}
		case command:
			return args[index : index+1], true
		default:
			return nil, true
		}
	}

	return nil, true
}

// longFlags are the shells' long options that take no value; longValueOptions
// those taking the next word as value.
var (
	longFlags = []string{
		"--norc", "--noprofile", "--login", "--posix", "--noediting", "--verbose", "--restricted", "--debugger",
		"--dump-strings", "--no-config", "--private", "--interactive", "--no-execute", "--emulate-sh",
	}
	longValueOptions = []string{"--rcfile", "--init-file"}
)

// carriesCommand is whether word is a group of short options holding c.
func carriesCommand(word *syntax.Word) bool {
	value, _ := Literal(word)

	return len(value) > 1 && value[0] == '-' && value[1] != '-' && strings.Contains(value, "c")
}

// groupValues is how many of the next words a group of short options takes as
// values, and whether the shell named reads that for certain: bash's o and O
// each take one, zsh's o takes the rest of the group or else the next word,
// and any other shell's o takes the next word only where it ends the group.
func groupValues(shell string, letters string) (int, bool) {
	switch shell {
	case bashShell:
		return strings.Count(letters, "o") + strings.Count(letters, "O"), true
	case zshShell:
		at := strings.IndexByte(letters, 'o')
		if at < 0 || at < len(letters)-1 {
			return 0, true
		}

		return 1, true
	}

	at := strings.IndexAny(letters, "oO")

	switch {
	case at < 0:
		return 0, true
	case at == len(letters)-1 && letters[at] == 'o':
		return 1, true
	}

	return 0, false
}
