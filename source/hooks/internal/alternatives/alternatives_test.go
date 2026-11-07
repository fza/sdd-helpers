package alternatives_test

import (
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/alternatives"
)

func literal(words ...string) []alternatives.Arg {
	args := make([]alternatives.Arg, 0, len(words))
	for _, word := range words {
		args = append(args, alternatives.Arg{Text: word, Literal: true, OneWord: true})
	}

	return args
}

func TestOnlyTheExactAlarmFormLaunches(t *testing.T) {
	cases := []struct {
		name  string
		words []string
		at    int
	}{
		{"the alarm form", []string{"perl", "-e", "alarm 9; exec @ARGV", "make"}, 3},
		{"other perl code", []string{"perl", "-e", "print 1", "rm"}, -1},
		{"a fractional alarm", []string{"perl", "-e", "alarm 1.5; exec @ARGV", "make"}, -1},
		{"the form with nothing to run", []string{"perl", "-e", "alarm 9; exec @ARGV"}, -1},
		{"another program", []string{"ruby", "-e", "alarm 9; exec @ARGV", "make"}, -1},
		{"perl options after the form", []string{"perl", "-e", "alarm 9; exec @ARGV", "-e", "system 1", "x"}, -1},
		{"a program and its arguments", []string{"perl", "-e", "alarm 9; exec @ARGV", "make", "test"}, 3},
		{"shell code before another word", []string{"perl", "-e", "alarm 9; exec @ARGV", "rm -rf ~/x", "x"}, -1},
		{"one word with a space", []string{"perl", "-e", "alarm 9; exec @ARGV", "rm -rf x"}, -1},
		{"one word with a tab", []string{"perl", "-e", "alarm 9; exec @ARGV", "rm\tx"}, -1},
		{"one word with a semicolon", []string{"perl", "-e", "alarm 9; exec @ARGV", "make;ls"}, -1},
		{"one word with a dollar", []string{"perl", "-e", "alarm 9; exec @ARGV", "rm${IFS}x"}, -1},
		{"one word with a tilde", []string{"perl", "-e", "alarm 9; exec @ARGV", "~/bin/x"}, -1},
		{"one word with a newline", []string{"perl", "-e", "alarm 9; exec @ARGV", "a\nb"}, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := alternatives.PrefixLaunch(tc.words[0], literal(tc.words...)); got != tc.at {
				t.Errorf("only the exact alarm form should launch the word after it, got %d", got)
			}
		})
	}

	unknown := literal("perl", "-e", "alarm 9; exec @ARGV", "make")
	unknown[2].Literal = false

	if alternatives.PrefixLaunch("perl", unknown) >= 0 {
		t.Error("code the shell builds at run time should launch nothing")
	}

	single := literal("perl", "-e", "alarm 9; exec @ARGV", "make")
	single[3].Literal = false

	if alternatives.PrefixLaunch("perl", single) >= 0 {
		t.Error("a single word the shell builds at run time may hold shell code, and should launch nothing")
	}

	vanishing := literal("perl", "-e", "alarm 9; exec @ARGV", "make", "")
	vanishing[4] = alternatives.Arg{}

	if alternatives.PrefixLaunch("perl", vanishing) >= 0 {
		t.Error("a word after the program that may split or vanish should leave the call inline code")
	}

	quoted := literal("perl", "-e", "alarm 9; exec @ARGV", "make", "")
	quoted[4] = alternatives.Arg{OneWord: true}

	if alternatives.PrefixLaunch("perl", quoted) != 3 {
		t.Error("a word certain to stay one word, a double-quoted expansion among them, should keep the call a launcher")
	}
}
