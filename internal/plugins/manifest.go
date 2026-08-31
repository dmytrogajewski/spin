package plugins

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ParseManifest decodes plugin.json bytes against the closed Agent Plugins schema.
func ParseManifest(data []byte) (Manifest, error) {
	raw, err := decodeObject(data)
	if err != nil {
		return Manifest{}, err
	}

	unknown := collectUnknown(raw)
	manifest := Manifest{UnknownFields: unknown}

	if schemaErr := assignSchema(&manifest, raw); schemaErr != nil {
		return Manifest{}, schemaErr
	}

	if nameErr := assignName(&manifest, raw); nameErr != nil {
		return Manifest{}, nameErr
	}

	if metaErr := assignMetadata(&manifest, raw); metaErr != nil {
		return Manifest{}, metaErr
	}

	return manifest, nil
}

func decodeObject(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: top-level must be an object: %w", ErrInvalidManifest, err)
	}

	if raw == nil {
		return nil, fmt.Errorf("%w: top-level must be an object", ErrInvalidManifest)
	}

	return raw, nil
}

func collectUnknown(raw map[string]json.RawMessage) []string {
	unknown := make([]string, 0)

	for key := range raw {
		if _, ok := permittedFields[key]; ok {
			continue
		}

		unknown = append(unknown, key)
	}

	slices.Sort(unknown)

	return unknown
}

func assignSchema(manifest *Manifest, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldSchema]
	if !ok {
		return fmt.Errorf("%w: missing %s", ErrInvalidSchema, fieldSchema)
	}

	schema, err := decodeString(value, fieldSchema)
	if err != nil {
		return err
	}

	if schema != SchemaV1 {
		return fmt.Errorf("%w: %q", ErrInvalidSchema, schema)
	}

	manifest.Schema = schema

	return nil
}

func assignName(manifest *Manifest, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldName]
	if !ok {
		return fmt.Errorf("%w: missing %s", ErrInvalidName, fieldName)
	}

	name, err := decodeString(value, fieldName)
	if err != nil {
		return err
	}

	if nameErr := validateName(name); nameErr != nil {
		return nameErr
	}

	manifest.Name = name

	return nil
}

func assignMetadata(manifest *Manifest, raw map[string]json.RawMessage) error {
	if err := assignOptionalString(&manifest.Version, raw, fieldVersion); err != nil {
		return err
	}

	if err := assignOptionalString(&manifest.Description, raw, fieldDescription); err != nil {
		return err
	}

	if err := assignOptionalString(&manifest.Homepage, raw, fieldHomepage); err != nil {
		return err
	}

	if err := assignOptionalString(&manifest.Repository, raw, fieldRepository); err != nil {
		return err
	}

	if err := assignOptionalString(&manifest.License, raw, fieldLicense); err != nil {
		return err
	}

	if err := assignAuthor(manifest, raw); err != nil {
		return err
	}

	if err := assignKeywords(manifest, raw); err != nil {
		return err
	}

	return assignExtensions(manifest, raw)
}

func assignOptionalString(dst *string, raw map[string]json.RawMessage, field string) error {
	value, ok := raw[field]
	if !ok {
		return nil
	}

	text, err := decodeString(value, field)
	if err != nil {
		return err
	}

	*dst = text

	return nil
}

func assignAuthor(manifest *Manifest, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldAuthor]
	if !ok {
		return nil
	}

	author, err := decodeAuthor(value)
	if err != nil {
		return err
	}

	manifest.Author = author

	return nil
}

func assignKeywords(manifest *Manifest, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldKeywords]
	if !ok {
		return nil
	}

	var keywords []string
	if err := json.Unmarshal(value, &keywords); err != nil {
		return fmt.Errorf("%w: %s must be an array of strings", ErrInvalidField, fieldKeywords)
	}

	manifest.Keywords = keywords

	return nil
}

func assignExtensions(manifest *Manifest, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldExtensions]
	if !ok {
		return nil
	}

	obj, isObject := decodeObjectField(value)
	if !isObject {
		manifest.ExtensionsIgnored = true

		return nil
	}

	manifest.Extensions = obj

	return nil
}

func decodeObjectField(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false
	}

	if obj == nil {
		return nil, false
	}

	return obj, true
}

func decodeAuthor(raw json.RawMessage) (*Author, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, fmt.Errorf("%w: %s must be an object", ErrInvalidField, fieldAuthor)
	}

	for key := range obj {
		if !isAuthorField(key) {
			return nil, fmt.Errorf("%w: author.%s is not permitted", ErrInvalidField, key)
		}
	}

	author := &Author{}

	if err := assignOptionalString(&author.Name, obj, fieldAuthorName); err != nil {
		return nil, err
	}

	if err := assignOptionalString(&author.Email, obj, fieldAuthorEmail); err != nil {
		return nil, err
	}

	if err := assignOptionalString(&author.URL, obj, fieldAuthorURL); err != nil {
		return nil, err
	}

	return author, nil
}

func isAuthorField(key string) bool {
	return key == fieldAuthorName || key == fieldAuthorEmail || key == fieldAuthorURL
}

func decodeString(raw json.RawMessage, field string) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", fmt.Errorf("%w: %s must be a string", ErrInvalidField, field)
	}

	return text, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}

	if len(name) > MaxNameLen {
		return fmt.Errorf("%w: length %d exceeds %d", ErrInvalidName, len(name), MaxNameLen)
	}

	if !isAlphaNum(name[0]) {
		return fmt.Errorf("%w: must start with alphanumeric", ErrInvalidName)
	}

	if !isAlphaNum(name[len(name)-1]) {
		return fmt.Errorf("%w: must end with alphanumeric", ErrInvalidName)
	}

	if strings.Contains(name, "--") {
		return fmt.Errorf("%w: consecutive hyphens", ErrInvalidName)
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("%w: consecutive periods", ErrInvalidName)
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
	return isAlphaNum(ch) || ch == '-' || ch == '.'
}

func isAlphaNum(ch byte) bool {
	isLower := ch >= 'a' && ch <= 'z'
	isDigit := ch >= '0' && ch <= '9'

	return isLower || isDigit
}

func unknownFieldWarning(field string) string {
	return warnUnknownField + strconv.Quote(field)
}

func manifestWarnings(manifest Manifest) []string {
	warnings := make([]string, 0, len(manifest.UnknownFields)+1)

	for _, field := range manifest.UnknownFields {
		warnings = append(warnings, unknownFieldWarning(field))
	}

	if manifest.ExtensionsIgnored {
		warnings = append(warnings, warnExtensionsType)
	}

	return warnings
}
