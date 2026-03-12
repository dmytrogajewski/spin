package security

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cmdStr      string
		expectedCmd *Command
		expectError bool
	}{
		{
			name:   "simple command",
			cmdStr: "ls",
			expectedCmd: &Command{Program: "ls", Args: []string{}, Raw: "ls"},
		},
		{
			name:   "command with args",
			cmdStr: "ls -la /tmp",
			expectedCmd: &Command{Program: "ls", Args: []string{"-la", "/tmp"}, Raw: "ls -la /tmp"},
		},
		{
			name:   "command with multiple args",
			cmdStr: "git commit -m 'test message'",
			expectedCmd: &Command{Program: "git", Args: []string{"commit", "-m", "'test", "message'"}, Raw: "git commit -m 'test message'"},
		},
		{name: "empty command", cmdStr: "", expectedCmd: nil, expectError: true},
		{name: "whitespace only", cmdStr: "   \t\n  ", expectedCmd: nil, expectError: true},
		{
			name:   "uppercase program",
			cmdStr: "LS -LA",
			expectedCmd: &Command{Program: "ls", Args: []string{"-LA"}, Raw: "LS -LA"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := ParseCommand(tt.cmdStr)
			checkParseError(t, err, tt.expectError)
			assertCommandEquals(t, cmd, tt.expectedCmd)
		})
	}
}

func checkParseError(t *testing.T, err error, expectError bool) {
	t.Helper()
	if expectError && err == nil {
		t.Errorf("expected error but got none")
	}
	if !expectError && err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func assertCommandEquals(t *testing.T, got, want *Command) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
		return
	}

	if got == nil {
		t.Errorf("got nil, want %v", want)
		return
	}

	if got.Program != want.Program {
		t.Errorf("Program = %v, want %v", got.Program, want.Program)
	}
	if got.Raw != want.Raw {
		t.Errorf("Raw = %v, want %v", got.Raw, want.Raw)
	}
	if len(got.Args) != len(want.Args) {
		t.Errorf("Args length = %v, want %v", len(got.Args), len(want.Args))
		return
	}
	for i, arg := range got.Args {
		if arg != want.Args[i] {
			t.Errorf("Args[%d] = %v, want %v", i, arg, want.Args[i])
		}
	}
}

func TestCommandClass_String(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := tt.class.String()
			if result != tt.expected {
				t.Errorf("CommandClass.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommandClass_NeedsApproval(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := tt.class.NeedsApproval()
			if result != tt.expected {
				t.Errorf("CommandClass.NeedsApproval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_Classify(t *testing.T) {
	t.Parallel()
	validator := NewValidator()

	tests := []struct {
		name          string
		cmd           *Command
		expectedClass CommandClass
		expectError   bool
	}{
		{"safe command - ls", &Command{Program: "ls", Args: []string{"-la"}, Raw: "ls -la"}, CommandSafe, false},
		{"safe command - cat", &Command{Program: "cat", Args: []string{"file.txt"}, Raw: "cat file.txt"}, CommandSafe, false},
		{
			"interactive command - mkdir",
			&Command{Program: "mkdir", Args: []string{"newdir"}, Raw: "mkdir newdir"},
			CommandInteractive, false,
		},
		{
			"dangerous command - rm -rf",
			&Command{Program: "rm", Args: []string{"-rf", "test"}, Raw: "rm -rf test"},
			CommandDangerous, false,
		},
		{
			"forbidden command - rm -rf /",
			&Command{Program: "rm", Args: []string{"-rf", "/"}, Raw: "rm -rf /"},
			CommandForbidden, false,
		},
		{
			"unknown command",
			&Command{Program: "unknowncommand", Args: []string{"arg1"}, Raw: "unknowncommand arg1"},
			CommandUnverified, false,
		},
		{"nil command", nil, CommandUnverified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := validator.Classify(tt.cmd)
			checkParseError(t, err, tt.expectError)

			if result != nil && result.Classification != tt.expectedClass {
				t.Errorf("Classify().Classification = %v, want %v", result.Classification, tt.expectedClass)
			}
		})
	}
}

// validatorPredicateCase describes a test case for a Validator predicate method.
type validatorPredicateCase struct {
	name     string
	cmd      *Command
	expected bool
}

func runValidatorPredicateTests(t *testing.T, cases []validatorPredicateCase, opName string, op func(*Validator, *Command) bool) {
	t.Helper()
	validator := NewValidator()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := op(validator, tt.cmd)
			if result != tt.expected {
				t.Errorf("%s() = %v, want %v", opName, result, tt.expected)
			}
		})
	}
}

var (
	safeCmd = &Command{Program: "ls", Args: []string{"-la"}, Raw: "ls -la"}
	rmCmd   = &Command{Program: "rm", Args: []string{"-rf", "test"}, Raw: "rm -rf test"}
	rmRoot  = &Command{Program: "rm", Args: []string{"-rf", "/"}, Raw: "rm -rf /"}
)

func TestValidator_IsSafe(t *testing.T) {
	t.Parallel()
	runValidatorPredicateTests(t, []validatorPredicateCase{
		{"safe command", safeCmd, true},
		{"dangerous command", rmCmd, false},
		{"nil command", nil, false},
	}, "IsSafe", (*Validator).IsSafe)
}

func TestValidator_IsDangerous(t *testing.T) {
	t.Parallel()
	runValidatorPredicateTests(t, []validatorPredicateCase{
		{"safe command", safeCmd, false},
		{"dangerous command", rmCmd, true},
		{"nil command", nil, false},
	}, "IsDangerous", (*Validator).IsDangerous)
}

func TestValidator_IsForbidden(t *testing.T) {
	t.Parallel()
	runValidatorPredicateTests(t, []validatorPredicateCase{
		{"safe command", safeCmd, false},
		{"forbidden command", rmRoot, true},
		{"nil command", nil, false},
	}, "IsForbidden", (*Validator).IsForbidden)
}

func TestValidator_NeedsApproval(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := validator.NeedsApproval(tt.cmd)
			if result != tt.expected {
				t.Errorf("NeedsApproval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_SpecialForbiddenPatterns(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	validator := NewValidator()

	// Test concurrent classification.
	results := make(chan *ValidationResult, 10)
	errors := make(chan error, 10)

	for range 10 {
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

	// Collect results.
	for range 10 {
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
