package patchapply

import (
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Patch
		wantErr string
	}{
		// ========== Valid Patches ==========
		{
			name:  "empty patch",
			input: "*** Begin Patch\n*** End Patch\n",
			want:  &Patch{Operations: []FileOperation{}},
		},
		{
			name: "add file - simple",
			input: `*** Begin Patch
*** Add File: test.txt
+hello
+world
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "test.txt",
						Lines:    []string{"hello", "world"},
					},
				},
			},
		},
		{
			name: "add file - empty file",
			input: `*** Begin Patch
*** Add File: empty.txt
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "empty.txt",
						Lines:    []string{},
					},
				},
			},
		},
		{
			name: "add file - with empty lines",
			input: `*** Begin Patch
*** Add File: test.go
+package main
+
+func main() {}
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "test.go",
						Lines:    []string{"package main", "", "func main() {}"},
					},
				},
			},
		},
		{
			name: "delete file",
			input: `*** Begin Patch
*** Delete File: old.txt
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&DeleteFile{
						FilePath: "old.txt",
					},
				},
			},
		},
		{
			name: "update file - simple",
			input: `*** Begin Patch
*** Update File: main.go
@@
 func main() {
-    fmt.Println("old")
+    fmt.Println("new")
 }
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "main.go",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineContext, Text: "func main() {"},
								{Type: LineDelete, Text: "    fmt.Println(\"old\")"},
								{Type: LineInsert, Text: "    fmt.Println(\"new\")"},
								{Type: LineContext, Text: "}"},
							},
						}},
					},
				},
			},
		},
		{
			name: "update file - with context header",
			input: `*** Begin Patch
*** Update File: handler.go
@@ func (h *Handler) Process
 func (h *Handler) Process(data string) error {
-    return oldValue
+    return newValue
 }
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "handler.go",
						Hunks: []Hunk{{
							Header: "func (h *Handler) Process",
							Changes: []LineChange{
								{Type: LineContext, Text: "func (h *Handler) Process(data string) error {"},
								{Type: LineDelete, Text: "    return oldValue"},
								{Type: LineInsert, Text: "    return newValue"},
								{Type: LineContext, Text: "}"},
							},
						}},
					},
				},
			},
		},
		{
			name: "update file - move operation",
			input: `*** Begin Patch
*** Update File: old/path.go
*** Move to: new/path.go
@@
 package main
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "old/path.go",
						NewPath:  "new/path.go",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineContext, Text: "package main"},
							},
						}},
					},
				},
			},
		},
		{
			name: "update file - multiple hunks",
			input: `*** Begin Patch
*** Update File: multi.go
@@
-old1
+new1
@@
-old2
+new2
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "multi.go",
						Hunks: []Hunk{
							{
								Header: "",
								Changes: []LineChange{
									{Type: LineDelete, Text: "old1"},
									{Type: LineInsert, Text: "new1"},
								},
							},
							{
								Header: "",
								Changes: []LineChange{
									{Type: LineDelete, Text: "old2"},
									{Type: LineInsert, Text: "new2"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "update file - empty line in hunk",
			input: `*** Begin Patch
*** Update File: test.go
@@
 line1

-old
+new
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "test.go",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineContext, Text: "line1"},
								{Type: LineContext, Text: ""},
								{Type: LineDelete, Text: "old"},
								{Type: LineInsert, Text: "new"},
							},
						}},
					},
				},
			},
		},
		{
			name: "multiple operations",
			input: `*** Begin Patch
*** Add File: new.txt
+content
*** Delete File: old.txt
*** Update File: existing.txt
@@
-old
+new
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "new.txt",
						Lines:    []string{"content"},
					},
					&DeleteFile{
						FilePath: "old.txt",
					},
					&UpdateFile{
						FilePath: "existing.txt",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineDelete, Text: "old"},
								{Type: LineInsert, Text: "new"},
							},
						}},
					},
				},
			},
		},

		// ========== Syntax Errors ==========
		{
			name:    "missing begin marker",
			input:   "*** End Patch\n",
			wantErr: "expected '*** Begin Patch'",
		},
		{
			name:    "invalid begin marker",
			input:   "*** Begin Patchh\n*** End Patch\n",
			wantErr: "expected '*** Begin Patch'",
		},
		{
			name:    "missing end marker",
			input:   "*** Begin Patch\n",
			wantErr: "missing '*** End Patch'",
		},
		{
			name: "unknown operation",
			input: `*** Begin Patch
*** Unknown Operation: test.txt
*** End Patch`,
			wantErr: "unknown operation",
		},
		{
			name: "add file - invalid line prefix",
			input: `*** Begin Patch
*** Add File: test.txt
+line1
invalid line without prefix
+line2
*** End Patch`,
			wantErr: "invalid line format",
		},
		{
			name: "update file - invalid hunk line prefix",
			input: `*** Begin Patch
*** Update File: test.txt
@@
 context
invalid prefix
*** End Patch`,
			wantErr: "invalid line prefix",
		},

		// ========== Path Validation Errors ==========
		{
			name: "add file - absolute path",
			input: `*** Begin Patch
*** Add File: /etc/passwd
+malicious
*** End Patch`,
			wantErr: "absolute paths not allowed",
		},
		{
			name: "add file - path traversal",
			input: `*** Begin Patch
*** Add File: ../../../etc/passwd
+malicious
*** End Patch`,
			wantErr: "path traversal",
		},
		{
			name: "delete file - absolute path",
			input: `*** Begin Patch
*** Delete File: /etc/passwd
*** End Patch`,
			wantErr: "absolute paths not allowed",
		},
		{
			name: "delete file - path traversal",
			input: `*** Begin Patch
*** Delete File: ../../etc/passwd
*** End Patch`,
			wantErr: "path traversal",
		},
		{
			name: "update file - absolute path",
			input: `*** Begin Patch
*** Update File: /etc/passwd
@@
-old
+new
*** End Patch`,
			wantErr: "absolute paths not allowed",
		},
		{
			name: "update file - path traversal",
			input: `*** Begin Patch
*** Update File: ../../etc/passwd
@@
-old
+new
*** End Patch`,
			wantErr: "path traversal",
		},
		{
			name: "update file - move to absolute path",
			input: `*** Begin Patch
*** Update File: test.txt
*** Move to: /etc/passwd
@@
 content
*** End Patch`,
			wantErr: "absolute paths not allowed",
		},
		{
			name: "update file - move to path traversal",
			input: `*** Begin Patch
*** Update File: test.txt
*** Move to: ../../etc/passwd
@@
 content
*** End Patch`,
			wantErr: "path traversal",
		},

		// ========== Edge Cases ==========
		{
			name: "add file - nested path",
			input: `*** Begin Patch
*** Add File: src/internal/handler/new.go
+package handler
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "src/internal/handler/new.go",
						Lines:    []string{"package handler"},
					},
				},
			},
		},
		{
			name: "add file - unicode content",
			input: `*** Begin Patch
*** Add File: unicode.txt
+Hello 世界
+🚀 Emoji
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&AddFile{
						FilePath: "unicode.txt",
						Lines:    []string{"Hello 世界", "🚀 Emoji"},
					},
				},
			},
		},
		{
			name: "update file - only inserts",
			input: `*** Begin Patch
*** Update File: test.txt
@@
+new line 1
+new line 2
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "test.txt",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineInsert, Text: "new line 1"},
								{Type: LineInsert, Text: "new line 2"},
							},
						}},
					},
				},
			},
		},
		{
			name: "update file - only deletes",
			input: `*** Begin Patch
*** Update File: test.txt
@@
-old line 1
-old line 2
*** End Patch`,
			want: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "test.txt",
						Hunks: []Hunk{{
							Header: "",
							Changes: []LineChange{
								{Type: LineDelete, Text: "old line 1"},
								{Type: LineDelete, Text: "old line 2"},
							},
						}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			got, err := p.Parse()

			// Check error expectations
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("Parse() error = nil, wantErr %q", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Parse() error = %v, wantErr substring %q", err, tt.wantErr)
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}

			// Compare results
			if !equalPatch(got, tt.want) {
				t.Errorf("Parse() mismatch:\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

func TestParser_Parse_LineNumbers(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrMsg string // Expected error message including line number
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
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("LineChangeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare patches
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

// Helper function to compare operations
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
