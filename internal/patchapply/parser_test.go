package patchapply

import (
	"strings"
	"testing"
)

// parserTestCase defines a test case for parser tests.
type parserTestCase struct {
	name    string
	input   string
	want    *Patch
	wantErr string
}

// runParserTests runs a slice of parser test cases.
func runParserTests(t *testing.T, tests []parserTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			verifyParseResult(t, tt.input, tt.want, tt.wantErr)
		})
	}
}

// verifyParseResult parses input and verifies it matches expected result or error.
func verifyParseResult(t *testing.T, input string, want *Patch, wantErr string) {
	t.Helper()

	p := NewParser(input)
	got, err := p.Parse()

	if wantErr != "" {
		if err == nil {
			t.Errorf("Parse() error = nil, wantErr %q", wantErr)
			return
		}

		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("Parse() error = %v, wantErr substring %q", err, wantErr)
		}

		return
	}

	if err != nil {
		t.Errorf("Parse() unexpected error = %v", err)
		return
	}

	if !equalPatch(got, want) {
		t.Errorf("Parse() mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestParser_Parse_ValidPatches(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name:  "empty patch",
			input: "*** Begin Patch\n*** End Patch\n",
			want:  &Patch{Operations: []FileOperation{}},
		},
		{
			name: "add file - simple",
			input: "*** Begin Patch\n*** Add File: test.txt\n+hello\n+world\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "test.txt", Lines: []string{"hello", "world"}},
			}},
		},
		{
			name: "add file - empty file",
			input: "*** Begin Patch\n*** Add File: empty.txt\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "empty.txt", Lines: []string{}},
			}},
		},
		{
			name: "add file - with empty lines",
			input: "*** Begin Patch\n*** Add File: test.go\n+package main\n+\n+func main() {}\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "test.go", Lines: []string{"package main", "", "func main() {}"}},
			}},
		},
		{
			name: "delete file",
			input: "*** Begin Patch\n*** Delete File: old.txt\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&DeleteFile{FilePath: "old.txt"},
			}},
		},
	})
}

func TestParser_Parse_UpdateFile_Simple(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name: "update file - simple",
			input: "*** Begin Patch\n*** Update File: main.go\n@@\n func main() {\n" +
				"-    fmt.Println(\"old\")\n+    fmt.Println(\"new\")\n }\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&UpdateFile{FilePath: "main.go", Hunks: []Hunk{{
					Changes: []LineChange{
						{Type: LineContext, Text: "func main() {"},
						{Type: LineDelete, Text: "    fmt.Println(\"old\")"},
						{Type: LineInsert, Text: "    fmt.Println(\"new\")"},
						{Type: LineContext, Text: "}"},
					},
				}}},
			}},
		},
		{
			name: "update file - with context header",
			input: "*** Begin Patch\n*** Update File: handler.go\n" +
				"@@ func (h *Handler) Process\n func (h *Handler) Process(data string) error {\n" +
				"-    return oldValue\n+    return newValue\n }\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&UpdateFile{FilePath: "handler.go", Hunks: []Hunk{{
					Header: "func (h *Handler) Process",
					Changes: []LineChange{
						{Type: LineContext, Text: "func (h *Handler) Process(data string) error {"},
						{Type: LineDelete, Text: "    return oldValue"},
						{Type: LineInsert, Text: "    return newValue"},
						{Type: LineContext, Text: "}"},
					},
				}}},
			}},
		},
		{
			name:  "update file - move operation",
			input: "*** Begin Patch\n*** Update File: old/path.go\n*** Move to: new/path.go\n@@\n package main\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&UpdateFile{FilePath: "old/path.go", NewPath: "new/path.go", Hunks: []Hunk{{
					Changes: []LineChange{{Type: LineContext, Text: "package main"}},
				}}},
			}},
		},
	})
}

func TestParser_Parse_UpdateFile_Variants(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name:  "update file - multiple hunks",
			input: "*** Begin Patch\n*** Update File: multi.go\n@@\n-old1\n+new1\n@@\n-old2\n+new2\n*** End Patch",
			want: &Patch{Operations: []FileOperation{&UpdateFile{
				FilePath: "multi.go",
				Hunks: []Hunk{
					{Changes: []LineChange{{Type: LineDelete, Text: "old1"}, {Type: LineInsert, Text: "new1"}}},
					{Changes: []LineChange{{Type: LineDelete, Text: "old2"}, {Type: LineInsert, Text: "new2"}}},
				},
			}}},
		},
		{
			name:  "update file - empty line in hunk",
			input: "*** Begin Patch\n*** Update File: test.go\n@@\n line1\n\n-old\n+new\n*** End Patch",
			want: &Patch{Operations: []FileOperation{&UpdateFile{
				FilePath: "test.go",
				Hunks: []Hunk{{Changes: []LineChange{
					{Type: LineContext, Text: "line1"}, {Type: LineContext, Text: ""},
					{Type: LineDelete, Text: "old"}, {Type: LineInsert, Text: "new"},
				}}},
			}}},
		},
		{
			name:  "update file - only inserts",
			input: "*** Begin Patch\n*** Update File: test.txt\n@@\n+new line 1\n+new line 2\n*** End Patch",
			want: &Patch{Operations: []FileOperation{&UpdateFile{
				FilePath: "test.txt",
				Hunks: []Hunk{{Changes: []LineChange{
					{Type: LineInsert, Text: "new line 1"}, {Type: LineInsert, Text: "new line 2"},
				}}},
			}}},
		},
		{
			name:  "update file - only deletes",
			input: "*** Begin Patch\n*** Update File: test.txt\n@@\n-old line 1\n-old line 2\n*** End Patch",
			want: &Patch{Operations: []FileOperation{&UpdateFile{
				FilePath: "test.txt",
				Hunks: []Hunk{{Changes: []LineChange{
					{Type: LineDelete, Text: "old line 1"}, {Type: LineDelete, Text: "old line 2"},
				}}},
			}}},
		},
	})
}

func TestParser_Parse_MultipleOperations(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name: "multiple operations",
			input: "*** Begin Patch\n*** Add File: new.txt\n+content\n" +
				"*** Delete File: old.txt\n*** Update File: existing.txt\n@@\n-old\n+new\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "new.txt", Lines: []string{"content"}},
				&DeleteFile{FilePath: "old.txt"},
				&UpdateFile{FilePath: "existing.txt", Hunks: []Hunk{{
					Changes: []LineChange{{Type: LineDelete, Text: "old"}, {Type: LineInsert, Text: "new"}},
				}}},
			}},
		},
	})
}

func TestParser_Parse_SyntaxErrors(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{name: "missing begin marker", input: "*** End Patch\n", wantErr: "expected '*** Begin Patch'"},
		{name: "invalid begin marker", input: "*** Begin Patchh\n*** End Patch\n", wantErr: "expected '*** Begin Patch'"},
		{name: "missing end marker", input: "*** Begin Patch\n", wantErr: "missing '*** End Patch'"},
		{name: "unknown operation", input: "*** Begin Patch\n*** Unknown Operation: test.txt\n*** End Patch", wantErr: "unknown operation"},
		{
			name:    "add file - invalid line prefix",
			input:   "*** Begin Patch\n*** Add File: test.txt\n+line1\ninvalid line without prefix\n+line2\n*** End Patch",
			wantErr: "invalid line format",
		},
		{
			name:    "update file - invalid hunk line prefix",
			input:   "*** Begin Patch\n*** Update File: test.txt\n@@\n context\ninvalid prefix\n*** End Patch",
			wantErr: "invalid line prefix",
		},
	})
}

func TestParser_Parse_PathValidation(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name: "add file - absolute path", wantErr: "absolute paths not allowed",
			input: "*** Begin Patch\n*** Add File: /etc/passwd\n+malicious\n*** End Patch",
		},
		{
			name: "add file - path traversal", wantErr: "path traversal",
			input: "*** Begin Patch\n*** Add File: ../../../etc/passwd\n+malicious\n*** End Patch",
		},
		{
			name: "delete file - absolute path", wantErr: "absolute paths not allowed",
			input: "*** Begin Patch\n*** Delete File: /etc/passwd\n*** End Patch",
		},
		{
			name: "delete file - path traversal", wantErr: "path traversal",
			input: "*** Begin Patch\n*** Delete File: ../../etc/passwd\n*** End Patch",
		},
		{
			name: "update file - absolute path", wantErr: "absolute paths not allowed",
			input: "*** Begin Patch\n*** Update File: /etc/passwd\n@@\n-old\n+new\n*** End Patch",
		},
		{
			name: "update file - path traversal", wantErr: "path traversal",
			input: "*** Begin Patch\n*** Update File: ../../etc/passwd\n@@\n-old\n+new\n*** End Patch",
		},
		{
			name: "move to absolute path", wantErr: "absolute paths not allowed",
			input: "*** Begin Patch\n*** Update File: test.txt\n*** Move to: /etc/passwd\n@@\n content\n*** End Patch",
		},
		{
			name: "move to path traversal", wantErr: "path traversal",
			input: "*** Begin Patch\n*** Update File: test.txt\n*** Move to: ../../etc/passwd\n@@\n content\n*** End Patch",
		},
	})
}

func TestParser_Parse_EdgeCases(t *testing.T) {
	t.Parallel()

	runParserTests(t, []parserTestCase{
		{
			name: "add file - nested path",
			input: "*** Begin Patch\n*** Add File: src/internal/handler/new.go\n+package handler\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "src/internal/handler/new.go", Lines: []string{"package handler"}},
			}},
		},
		{
			name: "add file - unicode content",
			input: "*** Begin Patch\n*** Add File: unicode.txt\n+Hello \u4e16\u754c\n+\U0001f680 Emoji\n*** End Patch",
			want: &Patch{Operations: []FileOperation{
				&AddFile{FilePath: "unicode.txt", Lines: []string{"Hello \u4e16\u754c", "\U0001f680 Emoji"}},
			}},
		},
	})
}

func TestParser_Parse_LineNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantErrMsg string // Expected error message including line number.
	}{
		{
			name: "add file invalid line at line 4",
			input: `*** Begin Patch
*** Add File: test.txt
+line 1
invalid line without prefix
+line 2
*** End Patch`,
			wantErrMsg: "line 4:",
		},
		{
			name: "unknown operation at line 2",
			input: `*** Begin Patch
*** Unknown Operation: test.txt
*** End Patch`,
			wantErrMsg: "line 2:",
		},
		{
			name: "invalid path at line 2",
			input: `*** Begin Patch
*** Add File: /etc/passwd
+content
*** End Patch`,
			wantErrMsg: "line 2:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewParser(tt.input)

			_, err := p.Parse()
			if err == nil {
				t.Errorf("Parse() error = nil, want error with %q", tt.wantErrMsg)

				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestLineChangeType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ  LineChangeType
		want string
	}{
		{LineContext, "context"},
		{LineDelete, "delete"},
		{LineInsert, "insert"},
		{LineChangeType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.typ.String(); got != tt.want {
				t.Errorf("LineChangeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare patches.
func equalPatch(a, b *Patch) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a.Operations) != len(b.Operations) {
		return false
	}

	for i := range a.Operations {
		if !equalOperation(a.Operations[i], b.Operations[i]) {
			return false
		}
	}

	return true
}

// Helper function to compare operations.
func equalOperation(a, b FileOperation) bool {
	switch a := a.(type) {
	case *AddFile:
		b, ok := b.(*AddFile)
		if !ok {
			return false
		}

		return equalAddFile(a, b)
	case *DeleteFile:
		b, ok := b.(*DeleteFile)
		if !ok {
			return false
		}

		return equalDeleteFile(a, b)
	case *UpdateFile:
		b, ok := b.(*UpdateFile)
		if !ok {
			return false
		}

		return equalUpdateFile(a, b)
	default:
		return false
	}
}

func equalAddFile(a, b *AddFile) bool {
	if a.FilePath != b.FilePath {
		return false
	}

	if len(a.Lines) != len(b.Lines) {
		return false
	}

	for i := range a.Lines {
		if a.Lines[i] != b.Lines[i] {
			return false
		}
	}

	return true
}

func equalDeleteFile(a, b *DeleteFile) bool {
	return a.FilePath == b.FilePath
}

func equalUpdateFile(a, b *UpdateFile) bool {
	if a.FilePath != b.FilePath || a.NewPath != b.NewPath {
		return false
	}

	if len(a.Hunks) != len(b.Hunks) {
		return false
	}

	for i := range a.Hunks {
		if !equalHunk(&a.Hunks[i], &b.Hunks[i]) {
			return false
		}
	}

	return true
}

func equalHunk(a, b *Hunk) bool {
	if a.Header != b.Header {
		return false
	}

	if len(a.Changes) != len(b.Changes) {
		return false
	}

	for i := range a.Changes {
		if a.Changes[i].Type != b.Changes[i].Type || a.Changes[i].Text != b.Changes[i].Text {
			return false
		}
	}

	return true
}
