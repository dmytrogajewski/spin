package task

import "testing"

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
