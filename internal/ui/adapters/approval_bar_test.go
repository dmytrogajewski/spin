package adapters

import (
	"context"
	"strings"
	"testing"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

func TestNextApprovalMode_Cycles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cur  string
		want string
	}{
		{"empty_defaults_to_ask_then_yolo", "", ApprovalModeYolo},
		{"ask_to_yolo", ApprovalModeAsk, ApprovalModeYolo},
		{"yolo_back_to_ask", ApprovalModeYolo, ApprovalModeAsk},
		{"unknown_treated_as_ask", "garbage", ApprovalModeYolo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := nextApprovalMode(tt.cur); got != tt.want {
				t.Errorf("nextApprovalMode(%q) = %q, want %q", tt.cur, got, tt.want)
			}
		})
	}
}

func approvalRequestForBar(raw, workDir string) safety.ApprovalRequest {
	return safety.ApprovalRequest{
		ID:      "req-1",
		Command: &safety.Command{Program: "sh", Raw: raw},
		WorkDir: workDir,
	}
}

func TestBuildApprovalBar_FitsWidth(t *testing.T) {
	t.Parallel()

	longCmd := "rm -rf " + strings.Repeat("/very/long/path/segment", 10)
	req := approvalRequestForBar(longCmd, "/tmp/some/workdir")

	const width = 80

	bar := buildApprovalBar(req, width)
	plain := textwidth.StripANSI(bar)

	if got := uniseg.StringWidth(plain); got > width {
		t.Errorf("bar width = %d, want <= %d (plain: %q)", got, width, plain)
	}

	if !strings.Contains(plain, "approve?") {
		t.Errorf("bar missing label: %q", plain)
	}

	if !strings.Contains(plain, "[a]once [s]session [g]always [d]eny [esc]") {
		t.Errorf("bar missing key hints: %q", plain)
	}
}

func TestBuildApprovalBar_DropsWorkDirWhenTight(t *testing.T) {
	t.Parallel()

	longCmd := strings.Repeat("x", 200)
	req := approvalRequestForBar(longCmd, "/tmp/some/workdir")

	bar := textwidth.StripANSI(buildApprovalBar(req, minApprovalBarWidth))

	if strings.Contains(bar, barWDLabel+"/tmp") {
		t.Errorf("tight bar should drop workdir segment: %q", bar)
	}
}

func TestBuildApprovalBar_IncludesWorkDirWhenRoomy(t *testing.T) {
	t.Parallel()

	req := approvalRequestForBar("ls -la", "/tmp/some/workdir")

	bar := textwidth.StripANSI(buildApprovalBar(req, 120))

	if !strings.Contains(bar, "in /tmp/some/workdir") {
		t.Errorf("roomy bar should include workdir segment: %q", bar)
	}
}

func TestSanitizeApprovalCommand_CollapsesWhitespace(t *testing.T) {
	t.Parallel()

	got := sanitizeApprovalCommand("echo hi\n  && ls\t-la")
	want := "echo hi && ls -la"

	if got != want {
		t.Errorf("sanitizeApprovalCommand() = %q, want %q", got, want)
	}
}

func TestCycleApprovalMode_TogglesAndUpdatesStatus(t *testing.T) {
	t.Parallel()

	s := newTestPureTTYSetup()

	if got := s.p.ApprovalMode(); got != ApprovalModeAsk {
		t.Fatalf("default approval mode = %q, want %q", got, ApprovalModeAsk)
	}

	if got := s.p.CycleApprovalMode(); got != ApprovalModeYolo {
		t.Fatalf("first cycle = %q, want %q", got, ApprovalModeYolo)
	}

	if got := s.statusManager.GetMetrics().ApprovalMode; got != ApprovalModeYolo {
		t.Errorf("status manager approval mode = %q, want %q", got, ApprovalModeYolo)
	}

	if got := s.p.CycleApprovalMode(); got != ApprovalModeAsk {
		t.Fatalf("second cycle = %q, want %q", got, ApprovalModeAsk)
	}
}

func TestShowApprovalDialog_YoloAutoApproves(t *testing.T) {
	t.Parallel()

	s := newTestPureTTYSetup()
	s.p.SetApprovalMode(ApprovalModeYolo)

	req := approvalRequestForBar("rm -rf /tmp/build", "/tmp")

	resp := s.p.ShowApprovalDialog(context.Background(), req)

	if !resp.Approved {
		t.Fatal("yolo mode should auto-approve")
	}

	if resp.Scope != safety.ScopeOnce {
		t.Errorf("yolo approval scope = %q, want %q", resp.Scope, safety.ScopeOnce)
	}

	if !strings.Contains(s.buf.String(), "yolo: auto-approved") {
		t.Errorf("transcript should trace yolo approval, got: %q", s.buf.String())
	}
}
