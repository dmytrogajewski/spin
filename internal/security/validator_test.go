package security

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmdStr      string
		expectedCmd *Command
		expectError bool
	}{
		{
			name:   "simple command",
			cmdStr: "ls",
			expectedCmd: &Command{
				Program: "ls",
				Args:    []string{},
				Raw:     "ls",
			},
			expectError: false,
		},
		{
			name:   "command with args",
			cmdStr: "ls -la /tmp",
			expectedCmd: &Command{
				Program: "ls",
				Args:    []string{"-la", "/tmp"},
				Raw:     "ls -la /tmp",
			},
			expectError: false,
		},
		{
			name:   "command with multiple args",
			cmdStr: "git commit -m 'test message'",
			expectedCmd: &Command{
				Program: "git",
				Args:    []string{"commit", "-m", "'test", "message'"},
				Raw:     "git commit -m 'test message'",
			},
			expectError: false,
		},
		{
			name:        "empty command",
			cmdStr:      "",
			expectedCmd: nil,
			expectError: true,
		},
		{
			name:        "whitespace only",
			cmdStr:      "   \t\n  ",
			expectedCmd: nil,
			expectError: true,
		},
		{
			name:   "uppercase program",
			cmdStr: "LS -LA",
			expectedCmd: &Command{
				Program: "ls",
				Args:    []string{"-LA"},
				Raw:     "LS -LA",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.cmdStr)

			if tt.expectError && err == nil {
				t.Errorf("ParseCommand() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ParseCommand() unexpected error: %v", err)
			}

			if tt.expectedCmd == nil {
				if cmd != nil {
					t.Errorf("ParseCommand() = %v, want %v", cmd, tt.expectedCmd)
				}
				return
			}

			if cmd == nil {
				t.Errorf("ParseCommand() = %v, want %v", cmd, tt.expectedCmd)
				return
			}

			if cmd.Program != tt.expectedCmd.Program {
				t.Errorf("ParseCommand().Program = %v, want %v", cmd.Program, tt.expectedCmd.Program)
			}

			if len(cmd.Args) != len(tt.expectedCmd.Args) {
				t.Errorf("ParseCommand().Args length = %v, want %v", len(cmd.Args), len(tt.expectedCmd.Args))
			} else {
				for i, arg := range cmd.Args {
					if arg != tt.expectedCmd.Args[i] {
						t.Errorf("ParseCommand().Args[%d] = %v, want %v", i, arg, tt.expectedCmd.Args[i])
					}
				}
			}

			if cmd.Raw != tt.expectedCmd.Raw {
				t.Errorf("ParseCommand().Raw = %v, want %v", cmd.Raw, tt.expectedCmd.Raw)
			}
		})
	}
}

func TestCommandClass_String(t *testing.T) {
	tests := []struct {
		name     string
		class    CommandClass
		expected string
	}{
		{"safe", CommandSafe, "safe"},
		{"interactive", CommandInteractive, "interactive"},
		{"dangerous", CommandDangerous, "dangerous"},
		{"forbidden", CommandForbidden, "forbidden"},
		{"unverified", CommandUnverified, "unverified"},
		{"unknown", CommandClass(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.class.String()
			if result != tt.expected {
				t.Errorf("CommandClass.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommandClass_NeedsApproval(t *testing.T) {
	tests := []struct {
		name     string
		class    CommandClass
		expected bool
	}{
		{"safe", CommandSafe, false},
		{"interactive", CommandInteractive, true},
		{"dangerous", CommandDangerous, true},
		{"forbidden", CommandForbidden, true},
		{"unverified", CommandUnverified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.class.NeedsApproval()
			if result != tt.expected {
				t.Errorf("CommandClass.NeedsApproval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_Classify(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name          string
		cmd           *Command
		expectedClass CommandClass
		expectError   bool
	}{
		{
			name: "safe command - ls",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			},
			expectedClass: CommandSafe,
			expectError:   false,
		},
		{
			name: "safe command - cat",
			cmd: &Command{
				Program: "cat",
				Args:    []string{"file.txt"},
				Raw:     "cat file.txt",
			},
			expectedClass: CommandSafe,
			expectError:   false,
		},
		{
			name: "interactive command - mkdir",
			cmd: &Command{
				Program: "mkdir",
				Args:    []string{"newdir"},
				Raw:     "mkdir newdir",
			},
			expectedClass: CommandInteractive,
			expectError:   false,
		},
		{
			name: "dangerous command - rm -rf",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "test"},
				Raw:     "rm -rf test",
			},
			expectedClass: CommandDangerous,
			expectError:   false,
		},
		{
			name: "forbidden command - rm -rf /",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "/"},
				Raw:     "rm -rf /",
			},
			expectedClass: CommandForbidden,
			expectError:   false,
		},
		{
			name: "unknown command",
			cmd: &Command{
				Program: "unknowncommand",
				Args:    []string{"arg1"},
				Raw:     "unknowncommand arg1",
			},
			expectedClass: CommandUnverified,
			expectError:   false,
		},
		{
			name:          "nil command",
			cmd:           nil,
			expectedClass: CommandUnverified,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Classify(tt.cmd)

			if tt.expectError && err == nil {
				t.Errorf("Classify() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Classify() unexpected error: %v", err)
			}

			if result != nil && result.Classification != tt.expectedClass {
				t.Errorf("Classify().Classification = %v, want %v", result.Classification, tt.expectedClass)
			}
		})
	}
}

func TestValidator_IsSafe(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		cmd      *Command
		expected bool
	}{
		{
			name: "safe command",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			},
			expected: true,
		},
		{
			name: "dangerous command",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "test"},
				Raw:     "rm -rf test",
			},
			expected: false,
		},
		{
			name:     "nil command",
			cmd:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsSafe(tt.cmd)
			if result != tt.expected {
				t.Errorf("IsSafe() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_IsDangerous(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		cmd      *Command
		expected bool
	}{
		{
			name: "safe command",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			},
			expected: false,
		},
		{
			name: "dangerous command",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "test"},
				Raw:     "rm -rf test",
			},
			expected: true,
		},
		{
			name:     "nil command",
			cmd:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsDangerous(tt.cmd)
			if result != tt.expected {
				t.Errorf("IsDangerous() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_IsForbidden(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		cmd      *Command
		expected bool
	}{
		{
			name: "safe command",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			},
			expected: false,
		},
		{
			name: "forbidden command",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "/"},
				Raw:     "rm -rf /",
			},
			expected: true,
		},
		{
			name:     "nil command",
			cmd:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsForbidden(tt.cmd)
			if result != tt.expected {
				t.Errorf("IsForbidden() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_NeedsApproval(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		cmd      *Command
		expected bool
	}{
		{
			name: "safe command - no approval needed",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			},
			expected: false,
		},
		{
			name: "interactive command - approval needed",
			cmd: &Command{
				Program: "mkdir",
				Args:    []string{"newdir"},
				Raw:     "mkdir newdir",
			},
			expected: true,
		},
		{
			name: "dangerous command - approval needed",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "test"},
				Raw:     "rm -rf test",
			},
			expected: true,
		},
		{
			name: "forbidden command - no approval (blocked)",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "/"},
				Raw:     "rm -rf /",
			},
			expected: false,
		},
		{
			name: "unknown command - approval needed",
			cmd: &Command{
				Program: "unknowncommand",
				Args:    []string{"arg1"},
				Raw:     "unknowncommand arg1",
			},
			expected: true,
		},
		{
			name:     "nil command - approval needed (err on side of caution)",
			cmd:      nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.NeedsApproval(tt.cmd)
			if result != tt.expected {
				t.Errorf("NeedsApproval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_SpecialForbiddenPatterns(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name          string
		cmd           *Command
		expectedClass CommandClass
	}{
		{
			name: "fork bomb",
			cmd: &Command{
				Program: ":()",
				Args:    []string{},
				Raw:     ":(){ :|:& };:",
			},
			expectedClass: CommandForbidden,
		},
		{
			name: "disk overwrite",
			cmd: &Command{
				Program: ">",
				Args:    []string{"/dev/sda"},
				Raw:     "> /dev/sda",
			},
			expectedClass: CommandForbidden,
		},
		{
			name: "mkfs command",
			cmd: &Command{
				Program: "mkfs.ext4",
				Args:    []string{"/dev/sda1"},
				Raw:     "mkfs.ext4 /dev/sda1",
			},
			expectedClass: CommandForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Classify(tt.cmd)
			if err != nil {
				t.Errorf("Classify() error = %v", err)
			}
			if result.Classification != tt.expectedClass {
				t.Errorf("Classify().Classification = %v, want %v", result.Classification, tt.expectedClass)
			}
		})
	}
}

func TestValidator_Concurrency(t *testing.T) {
	validator := NewValidator()

	// Test concurrent classification
	results := make(chan *ValidationResult, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			cmd := &Command{
				Program: "ls",
				Args:    []string{"-la"},
				Raw:     "ls -la",
			}
			result, err := validator.Classify(cmd)
			results <- result
			errors <- err
		}()
	}

	// Collect results
	for i := 0; i < 10; i++ {
		result := <-results
		err := <-errors

		if err != nil {
			t.Errorf("Concurrent Classify() error = %v", err)
		}
		if result.Classification != CommandSafe {
			t.Errorf("Concurrent Classify().Classification = %v, want %v", result.Classification, CommandSafe)
		}
	}
}
