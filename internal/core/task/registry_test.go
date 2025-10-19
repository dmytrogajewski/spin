package task

import (
	"errors"
	"testing"
)

// mockTask is a test implementation of Task interface
type mockTask struct {
	name          string
	systemPrompt  string
	allowedTools  []string
	maxTokens     int
	validateError error
}

func (m *mockTask) Name() string {
	return m.name
}

func (m *mockTask) SystemPrompt() string {
	return m.systemPrompt
}

func (m *mockTask) AllowedTools() []string {
	return m.allowedTools
}

func (m *mockTask) MaxTokens() int {
	return m.maxTokens
}

func (m *mockTask) Validate() error {
	return m.validateError
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.tasks == nil {
		t.Errorf("NewRegistry() tasks map is nil")
	}

	if len(registry.tasks) != 0 {
		t.Errorf("NewRegistry() tasks length = %d, want 0", len(registry.tasks))
	}

	if registry.defaultTask != "" {
		t.Errorf("NewRegistry() defaultTask = %s, want empty string", registry.defaultTask)
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errType error
	}{
		{
			name:    "register valid task",
			task:    &mockTask{name: "test-task", systemPrompt: "test prompt that is long enough to meet minimum requirements", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: false,
		},
		{
			name:    "register nil task",
			task:    nil,
			wantErr: true,
			errType: ErrNilTask,
		},
		{
			name:    "register empty name",
			task:    &mockTask{name: "", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name with uppercase",
			task:    &mockTask{name: "Test-Task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name with special chars",
			task:    &mockTask{name: "test@task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name starting with dash",
			task:    &mockTask{name: "-test-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name ending with dash",
			task:    &mockTask{name: "test-task-", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name with consecutive dashes",
			task:    &mockTask{name: "test--task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register invalid name with consecutive underscores",
			task:    &mockTask{name: "test__task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register name too long",
			task:    &mockTask{name: "a" + string(make([]byte, 50)), systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrInvalidTaskName,
		},
		{
			name:    "register duplicate task",
			task:    &mockTask{name: "test-task", systemPrompt: "test prompt that is long enough to meet minimum requirements", allowedTools: []string{"tool1"}, maxTokens: 1000},
			wantErr: true,
			errType: ErrTaskAlreadyRegistered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskName := tt.name
			if tt.task != nil {
				taskName = tt.task.Name()
			}
			err := registry.Register(taskName, tt.task)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Registry.Register() expected error, got nil")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Registry.Register() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Registry.Register() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	// Register a task
	task := &mockTask{name: "test-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000}
	err := registry.Register("test-task", task)
	if err != nil {
		t.Fatalf("Registry.Register() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
		errType  error
	}{
		{
			name:     "get existing task",
			taskName: "test-task",
			wantErr:  false,
		},
		{
			name:     "get non-existing task",
			taskName: "non-existing",
			wantErr:  true,
			errType:  ErrTaskNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.Get(tt.taskName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Registry.Get() expected error, got nil")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Registry.Get() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Registry.Get() unexpected error: %v", err)
				}
				if result != task {
					t.Errorf("Registry.Get() returned task = %v, want %v", result, task)
				}
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("Registry.List() empty registry length = %d, want 0", len(list))
	}

	// Register multiple tasks
	tasks := []Task{
		&mockTask{name: "z-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
		&mockTask{name: "a-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
		&mockTask{name: "m-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000},
	}

	for _, task := range tasks {
		err := registry.Register(task.Name(), task)
		if err != nil {
			t.Fatalf("Registry.Register() unexpected error: %v", err)
		}
	}

	// Test sorted list
	list = registry.List()
	if len(list) != 3 {
		t.Errorf("Registry.List() length = %d, want 3", len(list))
	}

	// Check sorting
	expected := []string{"a-task", "m-task", "z-task"}
	for i, name := range list {
		if name != expected[i] {
			t.Errorf("Registry.List() [%d] = %s, want %s", i, name, expected[i])
		}
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()

	// Register a task
	task := &mockTask{name: "test-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000}
	err := registry.Register("test-task", task)
	if err != nil {
		t.Fatalf("Registry.Register() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		want     bool
	}{
		{
			name:     "has existing task",
			taskName: "test-task",
			want:     true,
		},
		{
			name:     "has non-existing task",
			taskName: "non-existing",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.Has(tt.taskName)
			if got != tt.want {
				t.Errorf("Registry.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry_SetDefault(t *testing.T) {
	registry := NewRegistry()

	// Register a task
	task := &mockTask{name: "test-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000}
	err := registry.Register("test-task", task)
	if err != nil {
		t.Fatalf("Registry.Register() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
		errType  error
	}{
		{
			name:     "set default existing task",
			taskName: "test-task",
			wantErr:  false,
		},
		{
			name:     "set default non-existing task",
			taskName: "non-existing",
			wantErr:  true,
			errType:  ErrTaskNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.SetDefault(tt.taskName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Registry.SetDefault() expected error, got nil")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Registry.SetDefault() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Registry.SetDefault() unexpected error: %v", err)
				}
				if registry.defaultTask != tt.taskName {
					t.Errorf("Registry.SetDefault() defaultTask = %s, want %s", registry.defaultTask, tt.taskName)
				}
			}
		})
	}
}

func TestRegistry_GetDefault(t *testing.T) {
	registry := NewRegistry()

	// Test no default set
	_, err := registry.GetDefault()
	if err == nil {
		t.Errorf("Registry.GetDefault() expected error, got nil")
	}
	if !errors.Is(err, ErrNoDefaultTask) {
		t.Errorf("Registry.GetDefault() expected error type %v, got %v", ErrNoDefaultTask, err)
	}

	// Register and set default task
	task := &mockTask{name: "test-task", systemPrompt: "test prompt", allowedTools: []string{"tool1"}, maxTokens: 1000}
	err = registry.Register("test-task", task)
	if err != nil {
		t.Fatalf("Registry.Register() unexpected error: %v", err)
	}

	err = registry.SetDefault("test-task")
	if err != nil {
		t.Fatalf("Registry.SetDefault() unexpected error: %v", err)
	}

	// Test get default
	result, err := registry.GetDefault()
	if err != nil {
		t.Errorf("Registry.GetDefault() unexpected error: %v", err)
	}
	if result != task {
		t.Errorf("Registry.GetDefault() returned task = %v, want %v", result, task)
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			task := &mockTask{
				name:         "task" + string(rune('0'+i)),
				systemPrompt: "test prompt that is long enough to meet minimum requirements",
				allowedTools: []string{"tool1"},
				maxTokens:    1000,
			}
			registry.Register(task.Name(), task)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have registered all tasks
	list := registry.List()
	if len(list) != 10 {
		t.Errorf("Registry concurrent access, list length = %d, want 10", len(list))
	}
}

func TestValidateTaskName(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		wantErr  bool
	}{
		{
			name:     "valid lowercase alphanumeric",
			taskName: "testtask",
			wantErr:  false,
		},
		{
			name:     "valid with hyphens",
			taskName: "test-task",
			wantErr:  false,
		},
		{
			name:     "valid with underscores",
			taskName: "test_task",
			wantErr:  false,
		},
		{
			name:     "valid mixed",
			taskName: "test-task_123",
			wantErr:  false,
		},
		{
			name:     "valid single character",
			taskName: "a",
			wantErr:  false,
		},
		{
			name:     "empty name",
			taskName: "",
			wantErr:  true,
		},
		{
			name:     "uppercase letter",
			taskName: "Test-Task",
			wantErr:  true,
		},
		{
			name:     "special characters",
			taskName: "test@task",
			wantErr:  true,
		},
		{
			name:     "starts with dash",
			taskName: "-test-task",
			wantErr:  true,
		},
		{
			name:     "ends with dash",
			taskName: "test-task-",
			wantErr:  true,
		},
		{
			name:     "starts with underscore",
			taskName: "_test-task",
			wantErr:  true,
		},
		{
			name:     "ends with underscore",
			taskName: "test-task_",
			wantErr:  true,
		},
		{
			name:     "consecutive dashes",
			taskName: "test--task",
			wantErr:  true,
		},
		{
			name:     "consecutive underscores",
			taskName: "test__task",
			wantErr:  true,
		},
		{
			name:     "too long",
			taskName: "a" + string(make([]byte, 50)),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskName(tt.taskName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTaskName() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTaskName) {
					t.Errorf("validateTaskName() expected error type %v, got %v", ErrInvalidTaskName, err)
				}
			} else {
				if err != nil {
					t.Errorf("validateTaskName() unexpected error: %v", err)
				}
			}
		})
	}
}
