package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
	"gopkg.in/yaml.v3"
)

// SkillsTool implements skill invocation
type SkillsTool struct {
	BaseTool
	skillsDirectory string
	cache           map[string]string
	mu              sync.RWMutex
}

// NewSkillsTool creates a new SkillsTool
func NewSkillsTool(skillsDirectory string) *SkillsTool {
	return &SkillsTool{
		BaseTool:        NewBaseTool("skills", "Invoke a skill"),
		skillsDirectory: skillsDirectory,
		cache:           make(map[string]string),
	}
}

// SkillMetadata represents the YAML frontmatter in SKILL.md
type SkillMetadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	Tags        []string `yaml:"tags"`
}

func (s *SkillsTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error) {
	skillName, ok := args["skill_name"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "skill_name is required", nil)
	}

	return s.InvokeSkill(skillName)
}

// InvokeSkill loads and returns the skill content
func (s *SkillsTool) InvokeSkill(skillName string) (string, error) {
	// Check cache first
	s.mu.RLock()
	if content, ok := s.cache[skillName]; ok {
		s.mu.RUnlock()
		return content, nil
	}
	s.mu.RUnlock()

	// Find skill directory
	skillPath := filepath.Join(s.skillsDirectory, skillName)
	skillFile := filepath.Join(skillPath, "SKILL.md")

	// Check if skill exists
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		return "", apperrors.New(apperrors.ErrCodeSkillNotFound,
			fmt.Sprintf("skill '%s' not found", skillName), nil)
	}

	// Read skill file
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation,
			"failed to read skill file", err)
	}

	// Parse YAML frontmatter and content
	formatted, err := s.parseSkillFile(string(content))
	if err != nil {
		return "", err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[skillName] = formatted
	s.mu.Unlock()

	return formatted, nil
}

func (s *SkillsTool) parseSkillFile(content string) (string, error) {
	// Split frontmatter and instructions
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		// No frontmatter, return as-is
		return content, nil
	}

	// Parse YAML frontmatter
	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		// If parsing fails, return original content
		return content, nil
	}

	// Format output with metadata
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Skill: %s\n\n", metadata.Name))
	if metadata.Description != "" {
		result.WriteString(fmt.Sprintf("**Description:** %s\n\n", metadata.Description))
	}
	if metadata.Version != "" {
		result.WriteString(fmt.Sprintf("**Version:** %s\n\n", metadata.Version))
	}

	// Add instructions
	result.WriteString("## Instructions\n\n")
	result.WriteString(strings.TrimSpace(parts[2]))

	return result.String(), nil
}

// DiscoverSkills returns a list of available skills
func (s *SkillsTool) DiscoverSkills() ([]SkillMetadata, error) {
	var skills []SkillMetadata

	if s.skillsDirectory == "" {
		return skills, nil
	}

	entries, err := os.ReadDir(s.skillsDirectory)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeFileOperation,
			"failed to read skills directory", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(s.skillsDirectory, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); os.IsNotExist(err) {
			continue
		}

		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		// Parse frontmatter
		parts := strings.SplitN(string(content), "---", 3)
		if len(parts) < 3 {
			continue
		}

		var metadata SkillMetadata
		if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
			continue
		}

		skills = append(skills, metadata)
	}

	return skills, nil
}
