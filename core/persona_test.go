package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTruncateFile(t *testing.T) {
	t.Run("short content unchanged", func(t *testing.T) {
		content := "hello world"
		got := TruncateFile(content, 100)
		if got != content {
			t.Errorf("expected %q, got %q", content, got)
		}
	})

	t.Run("zero limit returns original", func(t *testing.T) {
		content := "hello world"
		got := TruncateFile(content, 0)
		if got != content {
			t.Errorf("expected %q, got %q", content, got)
		}
	})

	t.Run("truncates long content", func(t *testing.T) {
		content := strings.Repeat("x", 1000)
		got := TruncateFile(content, 100)
		if !strings.Contains(got, "[...truncated...]") {
			t.Error("expected truncation marker")
		}
		// head 70 + tail 20 + marker
		if len(got) > 150 {
			t.Errorf("truncated result too long: %d chars", len(got))
		}
	})

	t.Run("exact limit unchanged", func(t *testing.T) {
		content := strings.Repeat("a", 100)
		got := TruncateFile(content, 100)
		if got != content {
			t.Error("content at exact limit should not be truncated")
		}
	})
}

func TestBuildProjectContext(t *testing.T) {
	t.Run("empty prompts returns empty", func(t *testing.T) {
		got := BuildProjectContext("/dir", PersonaPrompts{}, 0)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("soul only includes soul guidance", func(t *testing.T) {
		p := PersonaPrompts{Soul: "Be kind and helpful."}
		got := BuildProjectContext("/my/workspace", p, 0)
		if !strings.Contains(got, "# Project Context") {
			t.Error("missing Project Context heading")
		}
		if !strings.Contains(got, "embody its persona and tone") {
			t.Error("missing soul guidance line")
		}
		if !strings.Contains(got, "## /my/workspace/SOUL.md") {
			t.Error("missing full-path heading for SOUL.md")
		}
		if !strings.Contains(got, "Be kind and helpful.") {
			t.Error("missing soul content")
		}
	})

	t.Run("multiple files in correct order", func(t *testing.T) {
		p := PersonaPrompts{
			Agents:   "agents content",
			Soul:     "soul content",
			Identity: "identity content",
			Memory:   "memory content",
		}
		got := BuildProjectContext("/w", p, 0)

		// Use heading pattern to avoid matching the guidance text
		agentsIdx := strings.Index(got, "## /w/AGENTS.md")
		soulIdx := strings.Index(got, "## /w/SOUL.md")
		identityIdx := strings.Index(got, "## /w/IDENTITY.md")
		memoryIdx := strings.Index(got, "## /w/MEMORY.md")

		if agentsIdx > soulIdx || soulIdx > identityIdx || identityIdx > memoryIdx {
			t.Errorf("files not in expected order: AGENTS(%d) → SOUL(%d) → IDENTITY(%d) → MEMORY(%d)",
				agentsIdx, soulIdx, identityIdx, memoryIdx)
		}
	})

	t.Run("skips empty files", func(t *testing.T) {
		p := PersonaPrompts{Identity: "I am a bot"}
		got := BuildProjectContext("/w", p, 0)
		if strings.Contains(got, "SOUL.md") {
			t.Error("should not contain SOUL.md heading when soul is empty")
		}
		if strings.Contains(got, "embody its persona") {
			t.Error("should not contain soul guidance when soul is empty")
		}
	})

	t.Run("uses full path in heading", func(t *testing.T) {
		p := PersonaPrompts{Tools: "my tools"}
		got := BuildProjectContext("/home/user/persona", p, 0)
		if !strings.Contains(got, "## /home/user/persona/TOOLS.md") {
			t.Error("heading should use full path")
		}
	})

	t.Run("empty dir uses filename only", func(t *testing.T) {
		p := PersonaPrompts{Tools: "my tools"}
		got := BuildProjectContext("", p, 0)
		if !strings.Contains(got, "## TOOLS.md") {
			t.Error("heading should use filename when dir is empty")
		}
	})
}

func TestLoadPersonaDir(t *testing.T) {
	t.Run("loads existing files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("be nice"), 0644)
		os.WriteFile(filepath.Join(dir, "IDENTITY.md"), []byte("  I am bot  "), 0644)

		p, err := LoadPersonaDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if p.Soul != "be nice" {
			t.Errorf("Soul = %q, want %q", p.Soul, "be nice")
		}
		if p.Identity != "I am bot" {
			t.Errorf("Identity = %q, want %q (should be trimmed)", p.Identity, "I am bot")
		}
	})

	t.Run("skips missing files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul"), 0644)

		p, err := LoadPersonaDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if p.Soul != "soul" {
			t.Errorf("Soul = %q", p.Soul)
		}
		if p.Agents != "" || p.Tools != "" || p.Identity != "" || p.User != "" || p.Memory != "" {
			t.Error("missing files should result in empty strings")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		p, err := LoadPersonaDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !p.IsEmpty() {
			t.Error("all fields should be empty for empty directory")
		}
	})
}
