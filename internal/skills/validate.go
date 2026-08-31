package skills

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Validate enforces Agent Skills name, description, and optional-field rules.
func Validate(skill Skill) error {
	if err := validateName(skill.Name); err != nil {
		return err
	}

	if err := validateDirMatch(skill); err != nil {
		return err
	}

	if err := validateDescription(skill.Description); err != nil {
		return err
	}

	return validateCompatibility(skill.Compatibility)
}

func validateDirMatch(skill Skill) error {
	if skill.Dir == "" {
		return nil
	}

	dirName := filepath.Base(filepath.Clean(skill.Dir))
	if dirName == skill.Name {
		return nil
	}

	return fmt.Errorf("%w: name %q directory %q", ErrNameMismatch, skill.Name, dirName)
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}

	if len(name) > MaxNameLen {
		return fmt.Errorf("%w: length %d exceeds %d", ErrInvalidName, len(name), MaxNameLen)
	}

	if name[0] == '-' {
		return fmt.Errorf("%w: leading hyphen", ErrInvalidName)
	}

	if name[len(name)-1] == '-' {
		return fmt.Errorf("%w: trailing hyphen", ErrInvalidName)
	}

	if strings.Contains(name, "--") {
		return fmt.Errorf("%w: consecutive hyphens", ErrInvalidName)
	}

	for i := range len(name) {
		if isNameByte(name[i]) {
			continue
		}

		return fmt.Errorf("%w: invalid character %q", ErrInvalidName, string(name[i]))
	}

	return nil
}

func isNameByte(ch byte) bool {
	isLower := ch >= 'a' && ch <= 'z'
	isDigit := ch >= '0' && ch <= '9'

	return isLower || isDigit || ch == '-'
}

func validateDescription(description string) error {
	if description == "" {
		return fmt.Errorf("%w: empty", ErrInvalidDescription)
	}

	count := utf8.RuneCountInString(description)
	if count > MaxDescriptionLen {
		return fmt.Errorf("%w: length %d exceeds %d", ErrInvalidDescription, count, MaxDescriptionLen)
	}

	return nil
}

func validateCompatibility(compatibility string) error {
	if compatibility == "" {
		return nil
	}

	count := utf8.RuneCountInString(compatibility)
	if count > MaxCompatibilityLen {
		return fmt.Errorf("%w: length %d exceeds %d", ErrInvalidCompatibility, count, MaxCompatibilityLen)
	}

	return nil
}
