package test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

// The Agent Skills specification (agentskills.io/specification): the
// frontmatter fields it defines, the name rule, the field limits, and the
// body length it recommends.
var (
	skillName       = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	skillLink       = regexp.MustCompile(`\]\(([^)\s#]+)`)
	skillSpecFields = []string{"name", "description", "license", "compatibility", "metadata", "allowed-tools"}
)

const (
	skillNameLimit          = 64
	skillDescriptionLimit   = 1024
	skillCompatibilityLimit = 500
	skillBodyLineLimit      = 500
)

// SpecBreaches lists, for every skill directory under root, each way its
// SKILL.md departs from the Agent Skills specification: a field outside it,
// a name breaking its rule or naming another directory, a description or
// compatibility past its limit, metadata that is no map of strings, a body
// past the line count it recommends, a link leaving the skill or naming
// nothing, and a file beside SKILL.md that nothing links to, which an agent
// would never find.
func SpecBreaches(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("listing the skills: %w", err)
	}

	var breaches []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		found, err := skillBreaches(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}

		for _, breach := range found {
			breaches = append(breaches, entry.Name()+": "+breach)
		}
	}

	slices.Sort(breaches)

	return breaches, nil
}

func skillBreaches(dir string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"no SKILL.md"}, nil
		}

		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	header, body, found := strings.Cut(strings.TrimPrefix(string(content), "---\n"), "\n---\n")
	if !strings.HasPrefix(string(content), "---\n") || !found {
		return []string{"no frontmatter between --- lines"}, nil
	}

	var front map[string]any

	err = yaml.Unmarshal([]byte(header), &front)
	if err != nil {
		return []string{fmt.Sprintf("frontmatter is no YAML map: %v", err)}, nil
	}

	breaches := fieldBreaches(front, filepath.Base(dir))

	lines := strings.Count(body, "\n")
	if lines > skillBodyLineLimit {
		breaches = append(breaches, fmt.Sprintf("body of %d lines, past %d", lines, skillBodyLineLimit))
	}

	broken, linked := linkBreaches(dir, body)

	return append(breaches, append(broken, unlinkedFiles(dir, linked)...)...), nil
}

func fieldBreaches(front map[string]any, directory string) []string {
	var breaches []string

	for field := range front {
		if !slices.Contains(skillSpecFields, field) {
			breaches = append(breaches, fmt.Sprintf("field %q is outside the specification", field))
		}
	}

	name, _ := front["name"].(string)

	switch {
	case !skillName.MatchString(name) || utf8.RuneCountInString(name) > skillNameLimit:
		breaches = append(breaches, fmt.Sprintf("name %q breaks the name rule", name))
	case name != directory:
		breaches = append(breaches, fmt.Sprintf("name %q differs from its directory %q", name, directory))
	}

	description, _ := front["description"].(string)

	length := utf8.RuneCountInString(strings.TrimSpace(description))
	if length == 0 || length > skillDescriptionLimit {
		breaches = append(breaches, fmt.Sprintf("description of %d characters, outside 1 to %d", length, skillDescriptionLimit))
	}

	for _, field := range []string{"license", "compatibility", "allowed-tools"} {
		value, held := front[field]
		_, text := value.(string)

		if held && !text {
			breaches = append(breaches, fmt.Sprintf("field %q is no string", field))
		}
	}

	compatibility, _ := front["compatibility"].(string)
	if utf8.RuneCountInString(compatibility) > skillCompatibilityLimit {
		breaches = append(breaches, fmt.Sprintf("compatibility past %d characters", skillCompatibilityLimit))
	}

	metadata, held := front["metadata"]
	if held && !stringMap(metadata) {
		breaches = append(breaches, "metadata is no map of strings")
	}

	return breaches
}

func stringMap(value any) bool {
	entries, isMap := value.(map[string]any)
	if !isMap {
		return false
	}

	for _, entry := range entries {
		_, text := entry.(string)
		if !text {
			return false
		}
	}

	return true
}

// linkBreaches checks each relative link the body holds, and answers the
// files they name, relative to the skill.
func linkBreaches(dir string, body string) ([]string, map[string]bool) {
	var breaches []string

	linked := map[string]bool{}

	for _, match := range skillLink.FindAllStringSubmatch(body, -1) {
		link := match[1]
		if strings.Contains(link, "://") {
			continue
		}

		relative := filepath.Clean(link)
		if filepath.IsAbs(link) || relative == ".." || strings.HasPrefix(relative, "../") {
			breaches = append(breaches, fmt.Sprintf("link %s leaves the skill", link))

			continue
		}

		info, err := os.Stat(filepath.Join(dir, relative))
		if err != nil || info.IsDir() {
			breaches = append(breaches, fmt.Sprintf("link %s names no file in the skill", link))

			continue
		}

		linked[relative] = true
	}

	return breaches, linked
}

func unlinkedFiles(dir string, linked map[string]bool) []string {
	var breaches []string

	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		relative, _ := filepath.Rel(dir, path)
		if relative != "SKILL.md" && !linked[relative] {
			breaches = append(breaches, fmt.Sprintf("file %s is linked from nowhere in SKILL.md", relative))
		}

		return nil
	})

	return breaches
}
