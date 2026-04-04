package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PersonaPrompts holds workspace bootstrap file contents for system prompt injection.
// Field order matches the injection order (following OpenClaw convention).
type PersonaPrompts struct {
	Agents   string // AGENTS.md — operating instructions
	Soul     string // SOUL.md — persona & boundaries
	Tools    string // TOOLS.md — environment/tool notes
	Identity string // IDENTITY.md — agent self-description
	User     string // USER.md — user profile
	Memory   string // MEMORY.md — long-term curated memory
}

// IsEmpty returns true if all fields are empty.
func (p PersonaPrompts) IsEmpty() bool {
	return p.Agents == "" && p.Soul == "" && p.Tools == "" &&
		p.Identity == "" && p.User == "" && p.Memory == ""
}

const defaultPersonaMaxChars = 20000

// personaFiles defines the bootstrap files and their injection order.
var personaFiles = []struct {
	name  string
	field func(*PersonaPrompts) *string
}{
	{"AGENTS.md", func(p *PersonaPrompts) *string { return &p.Agents }},
	{"SOUL.md", func(p *PersonaPrompts) *string { return &p.Soul }},
	{"TOOLS.md", func(p *PersonaPrompts) *string { return &p.Tools }},
	{"IDENTITY.md", func(p *PersonaPrompts) *string { return &p.Identity }},
	{"USER.md", func(p *PersonaPrompts) *string { return &p.User }},
	{"MEMORY.md", func(p *PersonaPrompts) *string { return &p.Memory }},
}

// LoadPersonaDir reads workspace bootstrap files from the given directory.
// Missing files are silently skipped; other read errors are returned.
func LoadPersonaDir(dir string) (PersonaPrompts, error) {
	var p PersonaPrompts
	for _, f := range personaFiles {
		data, err := os.ReadFile(filepath.Join(dir, f.name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return p, fmt.Errorf("persona: read %s: %w", f.name, err)
		}
		*f.field(&p) = strings.TrimSpace(string(data))
	}
	return p, nil
}

// TruncateFile truncates content to maxChars using a head(70%) + tail(20%) strategy.
// Returns the original content if it fits within the limit.
func TruncateFile(content string, maxChars int) string {
	if maxChars <= 0 || len(content) <= maxChars {
		return content
	}
	head := maxChars * 70 / 100
	tail := maxChars * 20 / 100
	return content[:head] + "\n\n[...truncated...]\n\n" + content[len(content)-tail:]
}

// BuildProjectContext assembles the "# Project Context" system prompt section
// from the persona prompts. Section headings use full file paths (dir + filename)
// so users know where to edit them.
func BuildProjectContext(dir string, p PersonaPrompts, maxCharsPerFile int) string {
	if p.IsEmpty() {
		return ""
	}
	if maxCharsPerFile <= 0 {
		maxCharsPerFile = defaultPersonaMaxChars
	}

	type entry struct {
		name    string
		content string
	}
	files := []entry{
		{"AGENTS.md", p.Agents},
		{"SOUL.md", p.Soul},
		{"TOOLS.md", p.Tools},
		{"IDENTITY.md", p.Identity},
		{"USER.md", p.User},
		{"MEMORY.md", p.Memory},
	}

	var b strings.Builder
	b.WriteString("\n# Project Context\n\n")
	b.WriteString("The following workspace files have been loaded to provide persona and context.\n")
	if p.Soul != "" {
		b.WriteString("If SOUL.md is present, embody its persona and tone. Avoid stiff, generic replies.\n")
	}

	for _, f := range files {
		if f.content == "" {
			continue
		}
		heading := f.name
		if dir != "" {
			heading = filepath.Join(dir, f.name)
		}
		b.WriteString("\n## ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		b.WriteString(TruncateFile(f.content, maxCharsPerFile))
		b.WriteString("\n")
	}

	return b.String()
}
