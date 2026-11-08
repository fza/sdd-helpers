package grounding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const skillsRoot = "../../../../plugin/skills"

// skillText is a skill's whole text: SKILL.md and every reference file it
// links, where a refusal the skill quotes may live.
func skillText(t *testing.T, skill string) string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(skillsRoot, skill, "references", "*.md"))
	if err != nil {
		t.Fatalf("listing the skill's references: %v", err)
	}

	var text strings.Builder

	for _, path := range append([]string{filepath.Join(skillsRoot, skill, "SKILL.md")}, paths...) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		text.Write(content)
	}

	return text.String()
}

func TestTheSkillQuotesEveryCredentialRefusal(t *testing.T) {
	skill := skillText(t, "grounding")

	for name, placeholders := range map[string][]string{
		"credential.txt":             {"{rules}", "<rule ids>", "{where}", "<where>", "{line}", "<n>"},
		"credential-unreadable.txt":  {"{where}", "<file>"},
		"credential-stdin.txt":       {"{name}", "<name>"},
		"credential-no-draft.txt":    {},
		"credential-unresolved.txt":  {"{where}", "<file>", "{why}", "<why>"},
		"credential-undecodable.txt": {"{tool}", "<tool>"},
		"credential-interrupted.txt": {},
	} {
		quoted := strings.TrimSpace(refusal(name, placeholders...))
		if !strings.Contains(skill, quoted) {
			t.Errorf("the skill should quote the refusal %q as the gate prints it", quoted)
		}
	}
}

func TestTheSkillsAndDocsQuoteTheStagedBodyRefusal(t *testing.T) {
	quoted, _, _ := strings.Cut(refusal("engine-handle.txt", "{handle}", "<handle>"), "\n")

	usage, err := os.ReadFile("../../../../docs/grounding-gate/usage.md")
	if err != nil {
		t.Fatalf("reading the usage docs: %v", err)
	}

	for name, text := range map[string]string{
		"the capture skill":            skillText(t, "capture"),
		"the grounding skill":          skillText(t, "grounding"),
		"docs/grounding-gate/usage.md": string(usage),
	} {
		if !strings.Contains(strings.Join(strings.Fields(text), " "), quoted) {
			t.Errorf("%s should quote the staged body refusal as the gate prints it", name)
		}
	}
}
