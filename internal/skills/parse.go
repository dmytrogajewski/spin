package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse loads SKILL.md from dir and returns a validated Skill.
func Parse(dir string) (Skill, error) {
	path := filepath.Join(filepath.Clean(dir), FileName)

	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return Skill{}, fmt.Errorf("%w: %s", ErrMissingFile, path)
		}

		return Skill{}, fmt.Errorf("read %s: %w", path, readErr)
	}

	skill, decodeErr := decodeSkill(string(raw), dir)
	if decodeErr != nil {
		return Skill{}, decodeErr
	}

	if validateErr := Validate(skill); validateErr != nil {
		return Skill{}, validateErr
	}

	return skill, nil
}

func decodeSkill(content, dir string) (Skill, error) {
	yamlText, body, err := splitFrontmatter(content)
	if err != nil {
		return Skill{}, err
	}

	fields, err := decodeFields(yamlText)
	if err != nil {
		return Skill{}, err
	}

	return Skill{
		Name:          fields.name,
		Description:   fields.description,
		License:       fields.license,
		Compatibility: fields.compatibility,
		Metadata:      fields.metadata,
		AllowedTools:  fields.allowedTools,
		Body:          body,
		Dir:           dir,
	}, nil
}

func splitFrontmatter(content string) (yamlText, body string, err error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	prefix := frontmatterFence + "\n"

	if !strings.HasPrefix(normalized, prefix) {
		return "", "", fmt.Errorf("%w: file must start with %s", ErrInvalidFrontmatter, frontmatterFence)
	}

	rest := normalized[len(prefix):]

	cutYAML, after, found := strings.Cut(rest, "\n"+frontmatterFence)
	if !found {
		return "", "", fmt.Errorf("%w: missing closing %s", ErrInvalidFrontmatter, frontmatterFence)
	}

	return cutYAML, strings.TrimPrefix(after, "\n"), nil
}

type rawFields struct {
	name          string
	description   string
	license       string
	compatibility string
	metadata      map[string]string
	allowedTools  string
}

func decodeFields(yamlText string) (rawFields, error) {
	var payload map[string]any
	if err := yaml.Unmarshal([]byte(yamlText), &payload); err != nil {
		return rawFields{}, fmt.Errorf("%w: %w", ErrInvalidFrontmatter, err)
	}

	if payload == nil {
		payload = map[string]any{}
	}

	name, err := optionalString(payload, fieldName)
	if err != nil {
		return rawFields{}, err
	}

	description, err := optionalString(payload, fieldDescription)
	if err != nil {
		return rawFields{}, err
	}

	license, err := optionalString(payload, fieldLicense)
	if err != nil {
		return rawFields{}, err
	}

	compatibility, err := optionalString(payload, fieldCompatibility)
	if err != nil {
		return rawFields{}, err
	}

	allowedTools, err := optionalString(payload, fieldAllowedTools)
	if err != nil {
		return rawFields{}, err
	}

	fields := rawFields{
		name:          name,
		description:   description,
		license:       license,
		compatibility: compatibility,
		allowedTools:  allowedTools,
	}

	if value, exists := payload[fieldMetadata]; exists && value != nil {
		parsed, metaErr := stringMap(value)
		if metaErr != nil {
			return rawFields{}, metaErr
		}

		fields.metadata = parsed
	}

	return fields, nil
}

func optionalString(payload map[string]any, key string) (string, error) {
	value, exists := payload[key]
	if !exists || value == nil {
		return "", nil
	}

	text, isString := value.(string)
	if !isString {
		return "", fmt.Errorf("%w: %s must be a string", ErrInvalidField, key)
	}

	return text, nil
}

func stringMap(value any) (map[string]string, error) {
	raw, isMap := value.(map[string]any)
	if !isMap {
		return nil, fmt.Errorf("%w: %s must be a string map", ErrInvalidMetadata, fieldMetadata)
	}

	out := make(map[string]string, len(raw))

	for mapKey, item := range raw {
		text, isString := item.(string)
		if !isString {
			return nil, fmt.Errorf("%w: %s.%s must be a string", ErrInvalidMetadata, fieldMetadata, mapKey)
		}

		out[mapKey] = text
	}

	return out, nil
}
