package blocks

import (
	"testing"
)

var executeMetaValidateCases = []struct {
	name        string
	meta        ExecuteMeta
	expectError bool
}{
	{
		name:        "valid execute meta",
		meta:        ExecuteMeta{Command: "ls -la", CWD: "/tmp", Impact: "low"},
		expectError: false,
	},
	{
		name:        "valid with optional fields",
		meta: ExecuteMeta{
			Command: "git status", CWD: "/home/user/project", Impact: "medium",
			TimeoutSec: 30, ExitCode: intPtr(0), DurationMS: int64Ptr(1500), LinesOut: intPtr(10),
		},
		expectError: false,
	},
	{name: "missing command", meta: ExecuteMeta{CWD: "/tmp", Impact: "low"}, expectError: true},
	{name: "missing cwd", meta: ExecuteMeta{Command: "ls -la", Impact: "low"}, expectError: true},
	{name: "invalid impact", meta: ExecuteMeta{Command: "ls -la", CWD: "/tmp", Impact: "invalid"}, expectError: true},
	{name: "negative exit code", meta: ExecuteMeta{Command: "ls -la", CWD: "/tmp", Impact: "low", ExitCode: intPtr(-1)}, expectError: true},
	{
		name: "negative duration", expectError: true,
		meta: ExecuteMeta{Command: "ls -la", CWD: "/tmp", Impact: "low", DurationMS: int64Ptr(-100)},
	},
	{name: "negative lines out", meta: ExecuteMeta{Command: "ls -la", CWD: "/tmp", Impact: "low", LinesOut: intPtr(-5)}, expectError: true},
}

func TestExecuteMeta_Validate(t *testing.T) {
	t.Parallel()

	for _, tt := range executeMetaValidateCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.meta.Validate()
			if tt.expectError && err == nil {
				t.Errorf("Validate() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestExecuteMeta_ImpactValues(t *testing.T) {
	t.Parallel()
	validImpacts := []string{"low", "medium", "high"}

	for _, impact := range validImpacts {
		meta := ExecuteMeta{
			Command: "test",
			CWD:     "/tmp",
			Impact:  impact,
		}

		err := meta.Validate()
		if err != nil {
			t.Errorf("Validate() error for impact %s: %v", impact, err)
		}
	}
}

func TestExecuteMeta_ZeroValues(t *testing.T) {
	t.Parallel(
	// Test that zero values are valid.
	)

	meta := ExecuteMeta{
		Command:    "test",
		CWD:        "/tmp",
		Impact:     "low",
		TimeoutSec: 0,
		ExitCode:   intPtr(0),
		DurationMS: int64Ptr(0),
		LinesOut:   intPtr(0),
	}

	err := meta.Validate()
	if err != nil {
		t.Errorf("Validate() error for zero values: %v", err)
	}
}

func TestExecuteMeta_Structure(t *testing.T) {
	t.Parallel()
	meta := ExecuteMeta{
		Command:    "git status",
		CWD:        "/home/user/project",
		Impact:     "medium",
		TimeoutSec: 30,
		ExitCode:   intPtr(0),
		DurationMS: int64Ptr(1500),
		LinesOut:   intPtr(15),
	}

	if meta.Command != "git status" {
		t.Errorf("ExecuteMeta.Command = %v, want %v", meta.Command, "git status")
	}

	if meta.CWD != "/home/user/project" {
		t.Errorf("ExecuteMeta.CWD = %v, want %v", meta.CWD, "/home/user/project")
	}

	if meta.Impact != "medium" {
		t.Errorf("ExecuteMeta.Impact = %v, want %v", meta.Impact, "medium")
	}

	if meta.TimeoutSec != 30 {
		t.Errorf("ExecuteMeta.TimeoutSec = %v, want %v", meta.TimeoutSec, 30)
	}

	if meta.ExitCode == nil || *meta.ExitCode != 0 {
		t.Errorf("ExecuteMeta.ExitCode = %v, want %v", meta.ExitCode, 0)
	}

	if meta.DurationMS == nil || *meta.DurationMS != 1500 {
		t.Errorf("ExecuteMeta.DurationMS = %v, want %v", meta.DurationMS, 1500)
	}

	if meta.LinesOut == nil || *meta.LinesOut != 15 {
		t.Errorf("ExecuteMeta.LinesOut = %v, want %v", meta.LinesOut, 15)
	}
}

func TestExecuteMeta_EmptyValues(t *testing.T) {
	t.Parallel()
	meta := ExecuteMeta{}

	if meta.Command != "" {
		t.Errorf("ExecuteMeta.Command = %v, want %v", meta.Command, "")
	}

	if meta.CWD != "" {
		t.Errorf("ExecuteMeta.CWD = %v, want %v", meta.CWD, "")
	}

	if meta.Impact != "" {
		t.Errorf("ExecuteMeta.Impact = %v, want %v", meta.Impact, "")
	}

	if meta.TimeoutSec != 0 {
		t.Errorf("ExecuteMeta.TimeoutSec = %v, want %v", meta.TimeoutSec, 0)
	}

	if meta.ExitCode != nil {
		t.Errorf("ExecuteMeta.ExitCode = %v, want %v", meta.ExitCode, nil)
	}

	if meta.DurationMS != nil {
		t.Errorf("ExecuteMeta.DurationMS = %v, want %v", meta.DurationMS, nil)
	}

	if meta.LinesOut != nil {
		t.Errorf("ExecuteMeta.LinesOut = %v, want %v", meta.LinesOut, nil)
	}
}

func TestExecuteMeta_OptionalFields(t *testing.T) {
	t.Parallel(
	// Test that optional fields can be nil.
	)

	meta := ExecuteMeta{
		Command: "test",
		CWD:     "/tmp",
		Impact:  "low",
		// ExitCode, DurationMS, LinesOut are nil.
	}

	err := meta.Validate()
	if err != nil {
		t.Errorf("Validate() error for nil optional fields: %v", err)
	}

	// Test that optional fields can be set.
	meta.ExitCode = intPtr(1)
	meta.DurationMS = int64Ptr(2000)
	meta.LinesOut = intPtr(5)

	err = meta.Validate()
	if err != nil {
		t.Errorf("Validate() error for set optional fields: %v", err)
	}
}

func TestExecuteMeta_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		meta        ExecuteMeta
		expectError bool
	}{
		{
			name: "maximum timeout",
			meta: ExecuteMeta{
				Command:    "test",
				CWD:        "/tmp",
				Impact:     "low",
				TimeoutSec: 3600,
			},
			expectError: false,
		},
		{
			name: "maximum duration",
			meta: ExecuteMeta{
				Command:    "test",
				CWD:        "/tmp",
				Impact:     "low",
				DurationMS: int64Ptr(3600000),
			},
			expectError: false,
		},
		{
			name: "maximum lines out",
			meta: ExecuteMeta{
				Command:  "test",
				CWD:      "/tmp",
				Impact:   "low",
				LinesOut: intPtr(10000),
			},
			expectError: false,
		},
		{
			name: "high impact",
			meta: ExecuteMeta{
				Command: "rm -rf /",
				CWD:     "/",
				Impact:  "high",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.meta.Validate()
			if tt.expectError && err == nil {
				t.Errorf("Validate() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// Helper functions for creating pointers.
func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
