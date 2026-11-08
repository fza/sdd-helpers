package plugin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/test"
)

const skillsRoot = pluginRoot + "/skills"

// skillBudgets caps each skill's SKILL.md in characters. A skill loads whole
// into context each time it is needed, so its growth is paid on every load;
// reference files beside it load only when their trigger shows up and carry
// no budget. Each figure sits a little above the size the skill had when its
// budget was set, so the tree passes with room for a sentence or two; raising
// one is a change of its own, stating what the added text earns.
var skillBudgets = []struct {
	skill  string
	budget int
	reason string
}{
	{"capture", 4300, "4.0k when set: which capture path, what an entry carries, the engine capture's project rules; sdd-capture itself lives in references/sdd-capture.md"},
	{"grounding", 4900, "4.6k when set: the three reads, read depths, body as lead, ref kinds, the gate's note; config keys and every other refusal live in references/"},
}

func budgetsBySkill() map[string]int {
	budgets := map[string]int{}
	for _, entry := range skillBudgets {
		budgets[entry.skill] = entry.budget
	}

	return budgets
}

func TestEverySkillFitsItsBudget(t *testing.T) {
	sizes, err := test.SkillSizes(skillsRoot)
	if err != nil {
		t.Fatalf("measuring the skills: %v", err)
	}

	for _, breach := range test.OverBudget(sizes, budgetsBySkill()) {
		t.Errorf("every skill's SKILL.md should fit a budget of its own: %s", breach)
	}
}

func TestASkillPaddedPastItsBudgetIsCaught(t *testing.T) {
	for _, entry := range skillBudgets {
		t.Run(entry.skill, func(t *testing.T) {
			padded := t.TempDir()

			err := test.CopyTree(skillsRoot, padded)
			if err != nil {
				t.Fatalf("copying the skills: %v", err)
			}

			path := filepath.Join(padded, entry.skill, "SKILL.md")

			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the copied skill: %v", err)
			}

			err = os.WriteFile(path, append(content, strings.Repeat("x", entry.budget)...), 0o644)
			if err != nil {
				t.Fatalf("padding the copied skill: %v", err)
			}

			sizes, err := test.SkillSizes(padded)
			if err != nil {
				t.Fatalf("measuring the padded skills: %v", err)
			}

			breaches := test.OverBudget(sizes, budgetsBySkill())
			if len(breaches) != 1 || !strings.HasPrefix(breaches[0], entry.skill+":") {
				t.Errorf("a skill padded past its budget should be the one breach reported, got: %v", breaches)
			}
		})
	}
}

func TestASkillWithoutABudgetIsCaught(t *testing.T) {
	padded := t.TempDir()

	err := test.CopyTree(skillsRoot, padded)
	if err != nil {
		t.Fatalf("copying the skills: %v", err)
	}

	err = os.MkdirAll(filepath.Join(padded, "new"), 0o755)
	if err != nil {
		t.Fatalf("creating a new skill: %v", err)
	}

	err = os.WriteFile(filepath.Join(padded, "new", "SKILL.md"), []byte("# New\n"), 0o644)
	if err != nil {
		t.Fatalf("writing a new skill: %v", err)
	}

	sizes, err := test.SkillSizes(padded)
	if err != nil {
		t.Fatalf("measuring the skills: %v", err)
	}

	breaches := test.OverBudget(sizes, budgetsBySkill())
	if len(breaches) != 1 || !strings.HasPrefix(breaches[0], "new:") || !strings.Contains(breaches[0], "no budget") {
		t.Errorf("a skill with no budget should be reported, got: %v", breaches)
	}
}

func TestABudgetForARemovedSkillIsCaught(t *testing.T) {
	padded := t.TempDir()

	err := test.CopyTree(skillsRoot, padded)
	if err != nil {
		t.Fatalf("copying the skills: %v", err)
	}

	err = os.RemoveAll(filepath.Join(padded, "grounding"))
	if err != nil {
		t.Fatalf("removing a copied skill: %v", err)
	}

	sizes, err := test.SkillSizes(padded)
	if err != nil {
		t.Fatalf("measuring the skills: %v", err)
	}

	breaches := test.OverBudget(sizes, budgetsBySkill())
	if len(breaches) != 1 || !strings.HasPrefix(breaches[0], "grounding:") {
		t.Errorf("a budget left for a removed skill should be reported, got: %v", breaches)
	}
}

func TestEveryReferenceFileIsNamedFromItsSkill(t *testing.T) {
	references, err := filepath.Glob(skillsRoot + "/*/*.md")
	if err != nil {
		t.Fatalf("listing the skills' files: %v", err)
	}

	for _, reference := range references {
		name := filepath.Base(reference)
		if name == "SKILL.md" {
			continue
		}

		core, err := os.ReadFile(filepath.Join(filepath.Dir(reference), "SKILL.md"))
		if err != nil {
			t.Fatalf("reading the skill beside %s: %v", reference, err)
		}

		if !strings.Contains(string(core), "]("+name) {
			t.Errorf("a reference file should be linked from its skill's SKILL.md, or no agent reads it: %s", reference)
		}
	}
}
