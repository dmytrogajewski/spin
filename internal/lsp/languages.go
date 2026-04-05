package lsp

import (
	"path/filepath"
	"strings"
)

// LanguageConfig describes how to start and interact with a language server.
type LanguageConfig struct {
	// ID is the LSP language identifier (e.g., "go", "python", "typescript").
	ID string

	// Extensions lists file extensions handled by this language (e.g., ".go", ".py").
	Extensions []string

	// ServerCommand is the binary name for the language server (e.g., "gopls").
	ServerCommand string

	// ServerArgs are additional arguments passed to the language server.
	ServerArgs []string

	// RootMarkers are files that indicate the project root (e.g., "go.mod", "package.json").
	RootMarkers []string
}

// builtinLanguages contains the default language configurations.
// Each entry maps a language to its server command and file associations.
var builtinLanguages = []LanguageConfig{
	{
		ID:            "go",
		Extensions:    []string{".go"},
		ServerCommand: "gopls",
		ServerArgs:    []string{"serve"},
		RootMarkers:   []string{"go.mod", "go.sum"},
	},
	{
		ID:            "python",
		Extensions:    []string{".py", ".pyi"},
		ServerCommand: "pylsp",
		RootMarkers:   []string{"pyproject.toml", "setup.py", "requirements.txt"},
	},
	{
		ID:            "typescript",
		Extensions:    []string{".ts", ".tsx"},
		ServerCommand: "typescript-language-server",
		ServerArgs:    []string{"--stdio"},
		RootMarkers:   []string{"tsconfig.json", "package.json"},
	},
	{
		ID:            "javascript",
		Extensions:    []string{".js", ".jsx", ".mjs", ".cjs"},
		ServerCommand: "typescript-language-server",
		ServerArgs:    []string{"--stdio"},
		RootMarkers:   []string{"package.json", "jsconfig.json"},
	},
	{
		ID:            "rust",
		Extensions:    []string{".rs"},
		ServerCommand: "rust-analyzer",
		RootMarkers:   []string{"Cargo.toml"},
	},
	{
		ID:            "java",
		Extensions:    []string{".java"},
		ServerCommand: "jdtls",
		RootMarkers:   []string{"pom.xml", "build.gradle"},
	},
	{
		ID:            "c",
		Extensions:    []string{".c", ".h"},
		ServerCommand: "clangd",
		RootMarkers:   []string{"compile_commands.json", "CMakeLists.txt", "Makefile"},
	},
	{
		ID:            "cpp",
		Extensions:    []string{".cpp", ".hpp", ".cc", ".cxx"},
		ServerCommand: "clangd",
		RootMarkers:   []string{"compile_commands.json", "CMakeLists.txt", "Makefile"},
	},
	{
		ID:            "ruby",
		Extensions:    []string{".rb"},
		ServerCommand: "solargraph",
		ServerArgs:    []string{"stdio"},
		RootMarkers:   []string{"Gemfile"},
	},
}

// extensionIndex maps file extensions to their LanguageConfig for fast lookup.
var extensionIndex map[string]*LanguageConfig

func init() {
	const extensionsPerLanguage = 2

	extensionIndex = make(map[string]*LanguageConfig, len(builtinLanguages)*extensionsPerLanguage)

	for idx := range builtinLanguages {
		lang := &builtinLanguages[idx]

		for _, ext := range lang.Extensions {
			extensionIndex[ext] = lang
		}
	}
}

// DetectLanguage returns the language configuration for the given file path
// based on its extension. Returns [ErrUnsupportedLanguage] if no match is found.
func DetectLanguage(filePath string) (LanguageConfig, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return LanguageConfig{}, ErrUnsupportedLanguage
	}

	lang, ok := extensionIndex[ext]
	if !ok {
		return LanguageConfig{}, ErrUnsupportedLanguage
	}

	return *lang, nil
}
