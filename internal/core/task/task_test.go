package task

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Mock task implementation for testing
type mockTask struct {
	name         string
	systemPrompt string
	allowedTools []string
	maxTokens    int
	validateErr  error
}

func (m *mockTask) Name() string           { return m.name }
func (m *mockTask) SystemPrompt() string   { return m.systemPrompt }
func (m *mockTask) AllowedTools() []string { return m.allowedTools }
func (m *mockTask) MaxTokens() int         { return m.maxTokens }
func (m *mockTask) Validate() error        { return m.validateErr }

// TestNewRegistry tests registry creation
func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if r.tasks == nil {
		t.Error("NewRegistry() did not initialize tasks map")
	}

	if len(r.List()) != 0 {
		t.Error("NewRegistry() should create empty registry")
	}
}

// TestRegistry_Register tests task registration
func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		task     Task
		wantErr  bool
		errType  error
	}{
		{
			name:     "valid task",
			taskName: "test",
			task: &mockTask{
				name:         "test",
				systemPrompt: "test prompt that is long enough",
				allowedTools: []string{"tool1"},
				maxTokens:    1000,
			},
			wantErr: false,
		},
		{
			name:     "nil task",
			taskName: "test",
			task:     nil,
			wantErr:  true,
			errType:  ErrNilTask,
		},
		{
			name:     "empty name",
			taskName: "",
			task:     &mockTask{name: ""},
			wantErr:  true,
			errType:  ErrInvalidTaskName,
		},
		{
			name:     "invalid name with uppercase",
			taskName: "TestTask",
			task:     &mockTask{name: "TestTask"},
			wantErr:  true,
			errType:  ErrInvalidTaskName,
		},
		{
			name:     "invalid name with space",
			taskName: "test task",
			task:     &mockTask{name: "test task"},
			wantErr:  true,
			errType:  ErrInvalidTaskName,
		},
		{
			name:     "valid name with dash",
			taskName: "test-task",
			task: &mockTask{
				name:         "test-task",
				systemPrompt: "test prompt that is long enough",
				allowedTools: []string{"tool1"},
				maxTokens:    1000,
			},
			wantErr: false,
		},
		{
			name:     "valid name with underscore",
			taskName: "test_task",
			task: &mockTask{
				name:         "test_task",
				systemPrompt: "test prompt that is long enough",
				allowedTools: []string{"tool1"},
				maxTokens:    1000,
			},
			wantErr: false,
		},
		{
			name:     "valid name with number",
			taskName: "test123",
			task: &mockTask{
				name:         "test123",
				systemPrompt: "test prompt that is long enough",
				allowedTools: []string{"tool1"},
				maxTokens:    1000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			err := r.Register(tt.taskName, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Register() error = %v, want %v", err, tt.errType)
				}
			}

			// If no error, verify task was registered
			if !tt.wantErr {
				if !r.Has(tt.taskName) {
					t.Errorf("Task %q was not registered", tt.taskName)
				}
			}
		})
	}
}

// TestRegistry_Register_Duplicate tests duplicate registration
func TestRegistry_Register_Duplicate(t *testing.T) {
	r := NewRegistry()
	task := &mockTask{
		name:         "test",
		systemPrompt: "test prompt that is long enough",
		allowedTools: []string{"tool1"},
		maxTokens:    1000,
	}

	// First registration should succeed
	if err := r.Register("test", task); err != nil {
		t.Fatalf("First Register() failed: %v", err)
	}

	// Second registration should fail
	err := r.Register("test", task)
	if err == nil {
		t.Error("Register() should fail on duplicate registration")
	}
	if !errors.Is(err, ErrTaskAlreadyRegistered) {
		t.Errorf("Register() error = %v, want ErrTaskAlreadyRegistered", err)
	}
}

// TestRegistry_Get tests task retrieval
func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	task := &mockTask{
		name:         "test",
		systemPrompt: "test prompt that is long enough",
		allowedTools: []string{"tool1"},
		maxTokens:    1000,
	}
	if err := r.Register("test", task); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		wantTask Task
		wantErr  bool
	}{
		{
			name:     "existing task",
			taskName: "test",
			wantTask: task,
			wantErr:  false,
		},
		{
			name:     "non-existent task",
			taskName: "nonexistent",
			wantTask: nil,
			wantErr:  true,
		},
		{
			name:     "empty name",
			taskName: "",
			wantTask: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Get(tt.taskName)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.wantTask {
				t.Errorf("Get() = %v, want %v", got, tt.wantTask)
			}

			if tt.wantErr && !errors.Is(err, ErrTaskNotFound) {
				t.Errorf("Get() error = %v, want ErrTaskNotFound", err)
			}
		})
	}
}

// TestRegistry_List tests task listing
func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	if list := r.List(); len(list) != 0 {
		t.Errorf("List() on empty registry = %v, want []", list)
	}

	// Add tasks in unsorted order
	if err := r.Register("zebra", &mockTask{
		name:         "zebra",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register(zebra) failed: %v", err)
	}
	if err := r.Register("alpha", &mockTask{
		name:         "alpha",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register(alpha) failed: %v", err)
	}
	if err := r.Register("beta", &mockTask{
		name:         "beta",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register(beta) failed: %v", err)
	}

	list := r.List()
	expected := []string{"alpha", "beta", "zebra"}

	if !reflect.DeepEqual(list, expected) {
		t.Errorf("List() = %v, want %v (sorted)", list, expected)
	}
}

// TestRegistry_Has tests task existence check
func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("test", &mockTask{
		name:         "test",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		want     bool
	}{
		{
			name:     "existing task",
			taskName: "test",
			want:     true,
		},
		{
			name:     "non-existent task",
			taskName: "nonexistent",
			want:     false,
		},
		{
			name:     "empty name",
			taskName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Has(tt.taskName); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.taskName, got, tt.want)
			}
		})
	}
}

// TestRegistry_SetDefault tests setting default task
func TestRegistry_SetDefault(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("test", &mockTask{
		name:         "test",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
	}{
		{
			name:     "set existing task as default",
			taskName: "test",
			wantErr:  false,
		},
		{
			name:     "set non-existent task as default",
			taskName: "nonexistent",
			wantErr:  true,
		},
		{
			name:     "set empty name as default",
			taskName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.SetDefault(tt.taskName)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetDefault() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify default was set
				task, err := r.GetDefault()
				if err != nil {
					t.Errorf("GetDefault() error after SetDefault() = %v", err)
				}
				if task.Name() != tt.taskName {
					t.Errorf("GetDefault() name = %v, want %v", task.Name(), tt.taskName)
				}
			}
		})
	}
}

// TestRegistry_GetDefault tests getting default task
func TestRegistry_GetDefault(t *testing.T) {
	r := NewRegistry()

	// No default set
	_, err := r.GetDefault()
	if err == nil {
		t.Error("GetDefault() should fail when no default is set")
	}
	if !errors.Is(err, ErrNoDefaultTask) {
		t.Errorf("GetDefault() error = %v, want ErrNoDefaultTask", err)
	}

	// Set default
	task := &mockTask{
		name:         "test",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}
	if err := r.Register("test", task); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if err := r.SetDefault("test"); err != nil {
		t.Fatalf("SetDefault() failed: %v", err)
	}

	// Get default
	got, err := r.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if got != task {
		t.Errorf("GetDefault() = %v, want %v", got, task)
	}
}

// TestRegistry_Concurrent tests concurrent operations
func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()

	// Test concurrent registrations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("task%d", n)
			_ = r.Register(name, &mockTask{
				name:         name,
				systemPrompt: "test prompt that is long enough",
				maxTokens:    1000,
			})
		}(i)
	}
	wg.Wait()

	// Test concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("task%d", n)
			_, _ = r.Get(name)
			_ = r.Has(name)
		}(i)
	}
	wg.Wait()

	// Test concurrent list operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.List()
		}()
	}
	wg.Wait()

	list := r.List()
	if len(list) != 100 {
		t.Errorf("List() after concurrent operations = %d tasks, want 100", len(list))
	}
}

// TestRegistry_ConcurrentReadWrite tests concurrent reads during writes
func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	r := NewRegistry()

	// Register initial task
	if err := r.Register("initial", &mockTask{
		name:         "initial",
		systemPrompt: "test prompt that is long enough",
		maxTokens:    1000,
	}); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("write%d", n)
			_ = r.Register(name, &mockTask{
				name:         name,
				systemPrompt: "test prompt that is long enough",
				maxTokens:    1000,
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.Get("initial")
			_ = r.Has("initial")
			_ = r.List()
		}()
	}

	wg.Wait()

	// Verify integrity
	if !r.Has("initial") {
		t.Error("Initial task was lost during concurrent operations")
	}

	list := r.List()
	if len(list) != 51 { // 1 initial + 50 writes
		t.Errorf("List() = %d tasks, want 51", len(list))
	}
}

// TestValidateTaskName tests task name validation
func TestValidateTaskName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "task", false},
		{"valid with dash", "my-task", false},
		{"valid with underscore", "my_task", false},
		{"valid with number", "task123", false},
		{"valid complex", "task-name_123", false},
		{"empty", "", true},
		{"uppercase", "Task", true},
		{"uppercase mixed", "myTask", true},
		{"space", "my task", true},
		{"special char", "task!", true},
		{"special char @", "task@name", true},
		{"dot", "task.name", true},
		{"slash", "task/name", true},
		{"too long", string(make([]byte, 51)), true},
		{"starts with dash", "-task", true},
		{"ends with dash", "task-", true},
		{"starts with underscore", "_task", true},
		{"double dash", "task--name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTaskName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkRegistry_Register(b *testing.B) {
	r := NewRegistry()
	task := &mockTask{
		name:         "bench",
		systemPrompt: "benchmark prompt that is long enough",
		maxTokens:    1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("task%d", i)
		_ = r.Register(name, task)
	}
}

func BenchmarkRegistry_Get(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("task%d", i)
		_ = r.Register(name, &mockTask{
			name:         name,
			systemPrompt: "benchmark prompt that is long enough",
			maxTokens:    1000,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("task50")
	}
}

func BenchmarkRegistry_List(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("task%d", i)
		_ = r.Register(name, &mockTask{
			name:         name,
			systemPrompt: "benchmark prompt that is long enough",
			maxTokens:    1000,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.List()
	}
}

func BenchmarkRegistry_Has(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("task%d", i)
		_ = r.Register(name, &mockTask{
			name:         name,
			systemPrompt: "benchmark prompt that is long enough",
			maxTokens:    1000,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Has("task50")
	}
}
