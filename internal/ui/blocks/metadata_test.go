package blocks

import (
	"testing"
)

func intPtr(i int) *int       { return &i }
func int64Ptr(i int64) *int64 { return &i }

func TestExecuteMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    *ExecuteMeta
		wantErr bool
	}{
		{
			name: "valid",
			meta: &ExecuteMeta{
				Command: "go test",
				CWD:     "/home/user",
				Impact:  "medium",
			},
			wantErr: false,
		},
		{
			name: "valid_with_optional",
			meta: &ExecuteMeta{
				Command:    "go test",
				CWD:        "/home/user",
				Impact:     "high",
				TimeoutSec: 600,
				ExitCode:   intPtr(0),
				DurationMS: int64Ptr(4200),
				LinesOut:   intPtr(54),
			},
			wantErr: false,
		},
		{
			name:    "missing_command",
			meta:    &ExecuteMeta{CWD: "/home/user", Impact: "low"},
			wantErr: true,
		},
		{
			name:    "missing_cwd",
			meta:    &ExecuteMeta{Command: "ls", Impact: "low"},
			wantErr: true,
		},
		{
			name:    "invalid_impact",
			meta:    &ExecuteMeta{Command: "ls", CWD: "/", Impact: "invalid"},
			wantErr: true,
		},
		{
			name: "negative_exit_code",
			meta: &ExecuteMeta{
				Command:  "ls",
				CWD:      "/",
				Impact:   "low",
				ExitCode: intPtr(-1),
			},
			wantErr: true,
		},
		{
			name: "negative_duration",
			meta: &ExecuteMeta{
				Command:    "ls",
				CWD:        "/",
				Impact:     "low",
				DurationMS: int64Ptr(-1),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteMeta.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    *ReadMeta
		wantErr bool
	}{
		{
			name:    "valid",
			meta:    &ReadMeta{File: "test.go"},
			wantErr: false,
		},
		{
			name:    "valid_with_offset_limit",
			meta:    &ReadMeta{File: "test.go", Offset: 10, Limit: 100},
			wantErr: false,
		},
		{
			name:    "missing_file",
			meta:    &ReadMeta{},
			wantErr: true,
		},
		{
			name:    "negative_offset",
			meta:    &ReadMeta{File: "test.go", Offset: -1},
			wantErr: true,
		},
		{
			name:    "negative_limit",
			meta:    &ReadMeta{File: "test.go", Limit: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMeta.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGrepMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    *GrepMeta
		wantErr bool
	}{
		{
			name:    "valid_content",
			meta:    &GrepMeta{Pattern: "test", Mode: "content"},
			wantErr: false,
		},
		{
			name:    "valid_files",
			meta:    &GrepMeta{Pattern: "test", Mode: "files_with_matches"},
			wantErr: false,
		},
		{
			name:    "valid_count",
			meta:    &GrepMeta{Pattern: "test", Mode: "count"},
			wantErr: false,
		},
		{
			name:    "valid_with_context",
			meta:    &GrepMeta{Pattern: "test", Mode: "content", Context: 3},
			wantErr: false,
		},
		{
			name:    "missing_pattern",
			meta:    &GrepMeta{Mode: "content"},
			wantErr: true,
		},
		{
			name:    "invalid_mode",
			meta:    &GrepMeta{Pattern: "test", Mode: "invalid"},
			wantErr: true,
		},
		{
			name:    "negative_context",
			meta:    &GrepMeta{Pattern: "test", Mode: "content", Context: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GrepMeta.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatchMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    *PatchMeta
		wantErr bool
	}{
		{
			name:    "valid_success",
			meta:    &PatchMeta{File: "test.go", Succeeded: true},
			wantErr: false,
		},
		{
			name:    "valid_with_stats",
			meta:    &PatchMeta{File: "test.go", Succeeded: true, LinesAdded: intPtr(10), LinesRemoved: intPtr(5)},
			wantErr: false,
		},
		{
			name:    "valid_failure",
			meta:    &PatchMeta{File: "test.go", Succeeded: false, ErrorMsg: "conflict"},
			wantErr: false,
		},
		{
			name:    "missing_file",
			meta:    &PatchMeta{Succeeded: true},
			wantErr: true,
		},
		{
			name:    "negative_lines_added",
			meta:    &PatchMeta{File: "test.go", Succeeded: true, LinesAdded: intPtr(-1)},
			wantErr: true,
		},
		{
			name:    "negative_lines_removed",
			meta:    &PatchMeta{File: "test.go", Succeeded: true, LinesRemoved: intPtr(-1)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PatchMeta.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlanMeta_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    *PlanMeta
		wantErr bool
	}{
		{
			name:    "valid_all_pending",
			meta:    &PlanMeta{Total: 5, Pending: 5, InProgress: 0, Completed: 0},
			wantErr: false,
		},
		{
			name:    "valid_mixed",
			meta:    &PlanMeta{Total: 10, Pending: 3, InProgress: 2, Completed: 5},
			wantErr: false,
		},
		{
			name:    "valid_all_completed",
			meta:    &PlanMeta{Total: 3, Pending: 0, InProgress: 0, Completed: 3},
			wantErr: false,
		},
		{
			name:    "negative_total",
			meta:    &PlanMeta{Total: -1, Pending: 0, InProgress: 0, Completed: 0},
			wantErr: true,
		},
		{
			name:    "negative_pending",
			meta:    &PlanMeta{Total: 5, Pending: -1, InProgress: 0, Completed: 0},
			wantErr: true,
		},
		{
			name:    "sum_mismatch_over",
			meta:    &PlanMeta{Total: 5, Pending: 3, InProgress: 2, Completed: 2},
			wantErr: true,
		},
		{
			name:    "sum_mismatch_under",
			meta:    &PlanMeta{Total: 10, Pending: 2, InProgress: 1, Completed: 2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PlanMeta.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseExecuteMeta(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	b.SetMeta("command", "go test")
	b.SetMeta("cwd", "/home/user")
	b.SetMeta("impact", "medium")
	b.SetMeta("exit_code", float64(0)) // JSON unmarshals numbers as float64
	b.SetMeta("duration_ms", float64(4200))
	b.SetMeta("lines_out", float64(54))

	meta, err := ParseExecuteMeta(b)
	if err != nil {
		t.Fatalf("ParseExecuteMeta() error = %v", err)
	}
	if meta.Command != "go test" {
		t.Errorf("Command = %v, want 'go test'", meta.Command)
	}
	if meta.CWD != "/home/user" {
		t.Errorf("CWD = %v, want '/home/user'", meta.CWD)
	}
	if meta.Impact != "medium" {
		t.Errorf("Impact = %v, want 'medium'", meta.Impact)
	}
	if *meta.ExitCode != 0 {
		t.Errorf("ExitCode = %v, want 0", *meta.ExitCode)
	}
}

func TestSetExecuteMeta(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	meta := &ExecuteMeta{
		Command:    "go test",
		CWD:        "/home/user",
		Impact:     "low",
		ExitCode:   intPtr(0),
		DurationMS: int64Ptr(1500),
		LinesOut:   intPtr(20),
	}

	err := SetExecuteMeta(b, meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta() error = %v", err)
	}

	// Verify metadata was set
	cmd, ok := b.GetMeta("command")
	if !ok || cmd != "go test" {
		t.Errorf("Meta['command'] = %v, want 'go test'", cmd)
	}
}

func TestSetExecuteMeta_Invalid(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	meta := &ExecuteMeta{
		Command: "go test",
		// Missing CWD - invalid
		Impact: "low",
	}

	err := SetExecuteMeta(b, meta)
	if err == nil {
		t.Error("SetExecuteMeta() expected error for invalid metadata")
	}
}

func TestParseReadMeta(t *testing.T) {
	b := NewBlock(BlockTypeRead)
	b.SetMeta("file", "test.go")
	b.SetMeta("offset", float64(10))
	b.SetMeta("limit", float64(100))

	meta, err := ParseReadMeta(b)
	if err != nil {
		t.Fatalf("ParseReadMeta() error = %v", err)
	}
	if meta.File != "test.go" {
		t.Errorf("File = %v, want 'test.go'", meta.File)
	}
	if meta.Offset != 10 {
		t.Errorf("Offset = %v, want 10", meta.Offset)
	}
	if meta.Limit != 100 {
		t.Errorf("Limit = %v, want 100", meta.Limit)
	}
}

func TestSetReadMeta(t *testing.T) {
	b := NewBlock(BlockTypeRead)
	meta := &ReadMeta{
		File:   "test.go",
		Offset: 5,
		Limit:  50,
	}

	err := SetReadMeta(b, meta)
	if err != nil {
		t.Fatalf("SetReadMeta() error = %v", err)
	}

	file, ok := b.GetMeta("file")
	if !ok || file != "test.go" {
		t.Errorf("Meta['file'] = %v, want 'test.go'", file)
	}
}

func TestSetReadMeta_Invalid(t *testing.T) {
	b := NewBlock(BlockTypeRead)
	meta := &ReadMeta{
		// Missing file - invalid
		Offset: 5,
	}

	err := SetReadMeta(b, meta)
	if err == nil {
		t.Error("SetReadMeta() expected error for invalid metadata")
	}
}

func TestParseGrepMeta(t *testing.T) {
	b := NewBlock(BlockTypeGrep)
	b.SetMeta("pattern", "test")
	b.SetMeta("mode", "content")
	b.SetMeta("context", float64(3))

	meta, err := ParseGrepMeta(b)
	if err != nil {
		t.Fatalf("ParseGrepMeta() error = %v", err)
	}
	if meta.Pattern != "test" {
		t.Errorf("Pattern = %v, want 'test'", meta.Pattern)
	}
	if meta.Mode != "content" {
		t.Errorf("Mode = %v, want 'content'", meta.Mode)
	}
	if meta.Context != 3 {
		t.Errorf("Context = %v, want 3", meta.Context)
	}
}

func TestSetGrepMeta(t *testing.T) {
	b := NewBlock(BlockTypeGrep)
	meta := &GrepMeta{
		Pattern: "error",
		Mode:    "files_with_matches",
		Context: 2,
	}

	err := SetGrepMeta(b, meta)
	if err != nil {
		t.Fatalf("SetGrepMeta() error = %v", err)
	}

	pattern, ok := b.GetMeta("pattern")
	if !ok || pattern != "error" {
		t.Errorf("Meta['pattern'] = %v, want 'error'", pattern)
	}
}

func TestSetGrepMeta_Invalid(t *testing.T) {
	b := NewBlock(BlockTypeGrep)
	meta := &GrepMeta{
		Pattern: "test",
		Mode:    "invalid_mode", // Invalid
	}

	err := SetGrepMeta(b, meta)
	if err == nil {
		t.Error("SetGrepMeta() expected error for invalid metadata")
	}
}

func TestParsePatchMeta(t *testing.T) {
	b := NewBlock(BlockTypeApplyPatch)
	b.SetMeta("file", "test.go")
	b.SetMeta("succeeded", true)
	b.SetMeta("lines_added", float64(10))
	b.SetMeta("lines_removed", float64(5))

	meta, err := ParsePatchMeta(b)
	if err != nil {
		t.Fatalf("ParsePatchMeta() error = %v", err)
	}
	if meta.File != "test.go" {
		t.Errorf("File = %v, want 'test.go'", meta.File)
	}
	if !meta.Succeeded {
		t.Error("Succeeded = false, want true")
	}
	if *meta.LinesAdded != 10 {
		t.Errorf("LinesAdded = %v, want 10", *meta.LinesAdded)
	}
}

func TestSetPatchMeta(t *testing.T) {
	b := NewBlock(BlockTypeApplyPatch)
	meta := &PatchMeta{
		File:         "test.go",
		Succeeded:    true,
		LinesAdded:   intPtr(8),
		LinesRemoved: intPtr(3),
	}

	err := SetPatchMeta(b, meta)
	if err != nil {
		t.Fatalf("SetPatchMeta() error = %v", err)
	}

	file, ok := b.GetMeta("file")
	if !ok || file != "test.go" {
		t.Errorf("Meta['file'] = %v, want 'test.go'", file)
	}
}

func TestSetPatchMeta_Invalid(t *testing.T) {
	b := NewBlock(BlockTypeApplyPatch)
	meta := &PatchMeta{
		// Missing file - invalid
		Succeeded: true,
	}

	err := SetPatchMeta(b, meta)
	if err == nil {
		t.Error("SetPatchMeta() expected error for invalid metadata")
	}
}

func TestParsePlanMeta(t *testing.T) {
	b := NewBlock(BlockTypePlan)
	b.SetMeta("total", float64(10))
	b.SetMeta("pending", float64(3))
	b.SetMeta("in_progress", float64(2))
	b.SetMeta("completed", float64(5))

	meta, err := ParsePlanMeta(b)
	if err != nil {
		t.Fatalf("ParsePlanMeta() error = %v", err)
	}
	if meta.Total != 10 {
		t.Errorf("Total = %v, want 10", meta.Total)
	}
	if meta.Pending != 3 {
		t.Errorf("Pending = %v, want 3", meta.Pending)
	}
}

func TestSetPlanMeta(t *testing.T) {
	b := NewBlock(BlockTypePlan)
	meta := &PlanMeta{
		Total:      7,
		Pending:    2,
		InProgress: 1,
		Completed:  4,
	}

	err := SetPlanMeta(b, meta)
	if err != nil {
		t.Fatalf("SetPlanMeta() error = %v", err)
	}

	total, ok := b.GetMeta("total")
	if !ok || total != float64(7) {
		t.Errorf("Meta['total'] = %v, want 7", total)
	}
}

func TestSetPlanMeta_Invalid(t *testing.T) {
	b := NewBlock(BlockTypePlan)
	meta := &PlanMeta{
		Total:      5,
		Pending:    2,
		InProgress: 1,
		Completed:  1, // Sum != Total (invalid)
	}

	err := SetPlanMeta(b, meta)
	if err == nil {
		t.Error("SetPlanMeta() expected error for invalid metadata")
	}
}
