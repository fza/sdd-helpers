package plugin_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/test"
)

func TestEverySkillMeetsTheAgentSkillsSpec(t *testing.T) {
	breaches, err := test.SpecBreaches(skillsRoot)
	if err != nil {
		t.Fatalf("reading the skills: %v", err)
	}

	for _, breach := range breaches {
		t.Errorf("every skill should meet the Agent Skills specification: %s", breach)
	}
}

func TestASkillBreakingTheSpecIsCaught(t *testing.T) {
	description := regexp.MustCompile(`(?m)^description: .*$`)

	cases := []struct {
		name    string
		change  func(t *testing.T, dir string, skill string) string
		reports string
	}{
		{
			name:    "a field outside the specification",
			change:  frontmatterAdds("sdd-version: 1.0.0"),
			reports: `field "sdd-version" is outside the specification`,
		},
		{
			name:    "an upper-case name",
			change:  replaces("name: capture", "name: Capture"),
			reports: `name "Capture" breaks the name rule`,
		},
		{
			name:    "a name with two hyphens in a row",
			change:  replaces("name: capture", "name: cap--ture"),
			reports: `name "cap--ture" breaks the name rule`,
		},
		{
			name:    "a name past 64 characters",
			change:  replaces("name: capture", "name: "+strings.Repeat("c", 65)),
			reports: "breaks the name rule",
		},
		{
			name:    "a name naming another directory",
			change:  replaces("name: capture", "name: capturing"),
			reports: `name "capturing" differs from its directory "capture"`,
		},
		{
			name: "an empty description",
			change: func(t *testing.T, _ string, skill string) string {
				return description.ReplaceAllString(skill, "description: ''")
			},
			reports: "description of 0 characters",
		},
		{
			name: "a description past 1024 characters",
			change: func(t *testing.T, _ string, skill string) string {
				return description.ReplaceAllString(skill, "description: "+strings.Repeat("d", 1025))
			},
			reports: "description of 1025 characters",
		},
		{
			name:    "a license that is no string",
			change:  replaces("license: MIT", "license: [MIT]"),
			reports: `field "license" is no string`,
		},
		{
			name:    "a compatibility past 500 characters",
			change:  frontmatterAdds("compatibility: " + strings.Repeat("c", 501)),
			reports: "compatibility past 500 characters",
		},
		{
			name:    "metadata that is no map of strings",
			change:  frontmatterAdds("metadata:\n  versions: [1, 2]"),
			reports: "metadata is no map of strings",
		},
		{
			name:    "a body past 500 lines",
			change:  appends(strings.Repeat("\n", 501)),
			reports: "past 500",
		},
		{
			name:    "a link leaving the skill",
			change:  appends("\n[grounding](../grounding/SKILL.md)\n"),
			reports: "link ../grounding/SKILL.md leaves the skill",
		},
		{
			name:    "a link naming nothing",
			change:  appends("\n[reference](references/missing.md)\n"),
			reports: "link references/missing.md names no file in the skill",
		},
		{
			name: "a file nothing links to",
			change: func(t *testing.T, dir string, skill string) string {
				err := os.MkdirAll(filepath.Join(dir, "references"), 0o755)
				if err == nil {
					err = os.WriteFile(filepath.Join(dir, "references", "extra.md"), []byte("extra\n"), 0o644)
				}

				if err != nil {
					t.Fatalf("adding a file to the copied skill: %v", err)
				}

				return skill
			},
			reports: "file references/extra.md is linked from nowhere",
		},
		{
			name:    "frontmatter that is no YAML",
			change:  frontmatterAdds("name: [capture"),
			reports: "frontmatter is no YAML map",
		},
		{
			name:    "no frontmatter",
			change:  replaces("---\n", ""),
			reports: "no frontmatter",
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			copied := t.TempDir()

			err := test.CopyTree(skillsRoot, copied)
			if err != nil {
				t.Fatalf("copying the skills: %v", err)
			}

			dir := filepath.Join(copied, "capture")
			path := filepath.Join(dir, "SKILL.md")

			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the copied skill: %v", err)
			}

			err = os.WriteFile(path, []byte(each.change(t, dir, string(content))), 0o644)
			if err != nil {
				t.Fatalf("changing the copied skill: %v", err)
			}

			breaches, err := test.SpecBreaches(copied)
			if err != nil {
				t.Fatalf("reading the changed skills: %v", err)
			}

			if len(breaches) != 1 || !strings.HasPrefix(breaches[0], "capture: ") || !strings.Contains(breaches[0], each.reports) {
				t.Errorf("the change should be the one breach reported, naming what it breaks, got: %v", breaches)
			}
		})
	}
}

func replaces(old string, replacement string) func(*testing.T, string, string) string {
	return func(t *testing.T, _ string, skill string) string {
		if !strings.Contains(skill, old) {
			t.Fatalf("the copied skill should hold %q to change", old)
		}

		return strings.Replace(skill, old, replacement, 1)
	}
}

func frontmatterAdds(field string) func(*testing.T, string, string) string {
	return replaces("---\n", "---\n"+field+"\n")
}

func appends(text string) func(*testing.T, string, string) string {
	return func(_ *testing.T, _ string, skill string) string {
		return skill + text
	}
}
