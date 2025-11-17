package task

import (
	"reflect"
	"testing"
)

func TestNewTask_Regular(t *testing.T) {
	task, err := NewTask("regular")
	if err != nil {
		t.Fatalf("NewTask(regular) error = %v", err)
	}
	if task.Name() != "regular" {
		t.Errorf("task.Name() = %s, want regular", task.Name())
	}
}

func TestNewTask_EmptyDefaultsToRegular(t *testing.T) {
	task, err := NewTask("")
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Name() != "regular" {
		t.Errorf("task.Name() = %s, want regular", task.Name())
	}
}

func TestNewTask_Review(t *testing.T) {
	task, err := NewTask("review")
	if err != nil {
		t.Fatalf("NewTask(review) error = %v", err)
	}
	if task.Name() != "review" {
		t.Errorf("task.Name() = %s, want review", task.Name())
	}
}

func TestNewTask_Compact(t *testing.T) {
	task, err := NewTask("compact")
	if err != nil {
		t.Fatalf("NewTask(compact) error = %v", err)
	}
	if task.Name() != "compact" {
		t.Errorf("task.Name() = %s, want compact", task.Name())
	}
}

func TestNewTask_Planning(t *testing.T) {
	task, err := NewTask("planning")
	if err != nil {
		t.Fatalf("NewTask(planning) error = %v", err)
	}
	if task.Name() != "planning" {
		t.Errorf("task.Name() = %s, want planning", task.Name())
	}
}

func TestNewTask_Unknown(t *testing.T) {
	_, err := NewTask("unknown")
	if err == nil {
		t.Fatal("NewTask(unknown) expected error, got nil")
	}
}

func TestDefaultTask(t *testing.T) {
	task := DefaultTask()
	if task == nil {
		t.Fatal("DefaultTask() returned nil")
	}
	if task.Name() != "regular" {
		t.Errorf("DefaultTask().Name() = %s, want regular", task.Name())
	}
}

// TestValidateMode tests task mode validation
func TestValidateMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"valid regular", "regular", false},
		{"valid review", "review", false},
		{"valid compact", "compact", false},
		{"valid planning", "planning", false},
		{"empty string", "", false}, // empty is valid (default)
		{"invalid mode", "invalid", true},
		{"unknown mode", "unknown", true},
		{"uppercase", "REGULAR", true}, // case-sensitive
		{"mixed case", "Regular", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMode(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMode(%q) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
			}
		})
	}
}

// TestValidModes tests that ValidModes contains all expected modes
func TestValidModes(t *testing.T) {
	expected := []string{"regular", "review", "compact", "planning"}
	if !reflect.DeepEqual(ValidModes, expected) {
		t.Errorf("ValidModes = %v, want %v", ValidModes, expected)
	}
}

// TestValidModes_ConsistentWithNewTask tests that ValidModes matches NewTask() behavior
func TestValidModes_ConsistentWithNewTask(t *testing.T) {
	// Test that all ValidModes can be created via NewTask
	for _, mode := range ValidModes {
		_, err := NewTask(mode)
		if err != nil {
			t.Errorf("NewTask(%q) failed for valid mode: %v", mode, err)
		}
	}

	// Test that all modes from NewTask are in ValidModes
	testModes := []string{"regular", "review", "compact", "planning", ""}
	for _, mode := range testModes {
		_, err := NewTask(mode)
		if err != nil {
			continue // skip invalid modes for NewTask
		}

		// Empty string is valid for NewTask (defaults to regular) but not in ValidModes
		if mode == "" {
			continue
		}

		found := false
		for _, validMode := range ValidModes {
			if mode == validMode {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("NewTask(%q) succeeds but mode not in ValidModes", mode)
		}
	}
}
