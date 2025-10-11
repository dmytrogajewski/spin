package blocks

import (
	"strings"
	"testing"
)

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  int
	}{
		{"Default width", 80, 80},
		{"Custom width", 120, 120},
		{"Zero width uses default", 0, 80},
		{"Negative width uses default", -10, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRenderer(tt.width)
			if r.width != tt.want {
				t.Errorf("NewRenderer(%d).width = %d, want %d", tt.width, r.width, tt.want)
			}
		})
	}
}

func TestRendererSetWidth(t *testing.T) {
	r := NewRenderer(80)

	r.SetWidth(120)
	if r.width != 120 {
		t.Errorf("After SetWidth(120), width = %d, want 120", r.width)
	}

	// Zero and negative widths should be ignored
	r.SetWidth(0)
	if r.width != 120 {
		t.Errorf("After SetWidth(0), width = %d, want 120 (unchanged)", r.width)
	}

	r.SetWidth(-10)
	if r.width != 120 {
		t.Errorf("After SetWidth(-10), width = %d, want 120 (unchanged)", r.width)
	}
}

func TestRenderNilBlock(t *testing.T) {
	r := NewRenderer(80)

	_, err := r.Render(nil)
	if err == nil {
		t.Error("Render(nil) should return error")
	}

	header := r.RenderHeader(nil)
	if header != "" {
		t.Error("RenderHeader(nil) should return empty string")
	}

	footer := r.RenderFooter(nil)
	if footer != "" {
		t.Error("RenderFooter(nil) should return empty string")
	}
}

func TestRenderExecuteBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeExecute)
	block.Title = "Run tests"
	block.Body = "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS"

	exitCode := 0
	lines := 3
	duration := int64(4200)

	meta := &ExecuteMeta{
		Command:    "go test -race ./...",
		CWD:        "/home/user/project",
		Impact:     "medium",
		ExitCode:   &exitCode,
		LinesOut:   &lines,
		DurationMS: &duration,
	}
	if err := SetExecuteMeta(block, meta); err != nil {
		t.Fatal(err)
	}

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Verify output contains key elements
	if !strings.Contains(output, "EXECUTE") {
		t.Error("Output should contain EXECUTE tag")
	}
	if !strings.Contains(output, "Run tests") {
		t.Error("Output should contain title")
	}
	if !strings.Contains(output, "=== RUN TestFoo") {
		t.Error("Output should contain body")
	}
	if !strings.Contains(output, "[exit: 0]") {
		t.Error("Output should contain exit code chip")
	}
	if !strings.Contains(output, "[out: 3 lines]") {
		t.Error("Output should contain lines count chip")
	}
	if !strings.Contains(output, "[dur: 4.2s]") {
		t.Error("Output should contain duration chip")
	}
	if !strings.Contains(output, "↳") {
		t.Error("Output should contain completion status line with arrow")
	}
	if !strings.Contains(output, "Exit code: 0") {
		t.Error("Output should contain exit code in completion status")
	}
}

func TestRenderPlanBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypePlan)
	block.Body = "• Task 1\n✓ Task 2\n◦ Task 3"

	meta := &PlanMeta{
		Total:      3,
		Pending:    1,
		InProgress: 0,
		Completed:  2,
	}
	if err := SetPlanMeta(block, meta); err != nil {
		t.Fatal(err)
	}

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "PLAN") {
		t.Error("Output should contain PLAN tag")
	}
	if !strings.Contains(output, "3 total") {
		t.Error("Output should contain plan metadata")
	}
	if !strings.Contains(output, "Task 1") {
		t.Error("Output should contain task 1")
	}
	if !strings.Contains(output, "Task 2") {
		t.Error("Output should contain task 2")
	}
}

func TestRenderReadBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeRead)
	block.Body = "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"

	meta := &ReadMeta{
		File:   "main.go",
		Offset: 0,
		Limit:  10,
	}
	if err := SetReadMeta(block, meta); err != nil {
		t.Fatal(err)
	}

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "READ") {
		t.Error("Output should contain READ tag")
	}
	if !strings.Contains(output, "main.go") {
		t.Error("Output should contain filename")
	}
	if !strings.Contains(output, "package main") {
		t.Error("Output should contain code body")
	}
	// Check for line numbers (should have │1, │2, etc.)
	if !strings.Contains(output, "│") {
		t.Error("Output should contain gutter separator")
	}
}

func TestRenderPatchBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeApplyPatch)
	block.Body = "@@ -42,6 +42,7 @@ func main() {\n func process() {\n     fmt.Println(\"processing\")\n+    log.Info(\"started\")\n }"

	added := 1
	meta := &PatchMeta{
		File:       "main.go",
		Succeeded:  true,
		LinesAdded: &added,
	}
	if err := SetPatchMeta(block, meta); err != nil {
		t.Fatal(err)
	}

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "WRITE") {
		t.Error("Output should contain WRITE tag")
	}
	if !strings.Contains(output, "main.go") {
		t.Error("Output should contain filename")
	}
	if !strings.Contains(output, "@@") {
		t.Error("Output should contain hunk header")
	}
	if !strings.Contains(output, "Succeeded") {
		t.Errorf("Output should contain success message. Got:\n%s", output)
	}
	if !strings.Contains(output, "(+1 added)") {
		t.Error("Output should contain lines added")
	}
}

func TestRenderErrorBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeError)
	block.Body = "Command failed: exit status 1\nstack trace line 1\nstack trace line 2"

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "ERROR") {
		t.Error("Output should contain ERROR tag")
	}
	if !strings.Contains(output, "Command failed") {
		t.Error("Output should contain error message")
	}
}

func TestRenderCollapsedBlock(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeExecute)
	block.Body = "some output"
	block.FoldState = FoldStateCollapsed

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "EXECUTE") {
		t.Error("Output should contain tag")
	}
	if !strings.Contains(output, "collapsed") {
		t.Error("Output should contain collapsed badge")
	}
	if strings.Contains(output, "some output") {
		t.Error("Output should NOT contain body when collapsed")
	}
}

func TestRenderEmptyBody(t *testing.T) {
	r := NewRenderer(80)

	block := NewBlock(BlockTypeNotice)
	block.Body = ""

	output, err := r.Render(block)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "NOTICE") {
		t.Error("Output should contain tag")
	}
}

func TestMidEllipsize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{
			name:     "Short string unchanged",
			input:    "short",
			maxWidth: 10,
			want:     "short",
		},
		{
			name:     "Long string truncated",
			input:    "very long filename.go",
			maxWidth: 15,
			want:     "very lon…ame.go", // 60/40 split: 8 chars left, 6 chars right, 1 ellipsis
		},
		{
			name:     "Exact length unchanged",
			input:    "exactlen",
			maxWidth: 8,
			want:     "exactlen",
		},
		{
			name:     "Very short maxWidth",
			input:    "verylongstring",
			maxWidth: 5,
			want:     "ve…ng",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := midEllipsize(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("midEllipsize(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

func TestCalcGutterWidth(t *testing.T) {
	tests := []struct {
		lineCount int
		want      int
	}{
		{5, 3},
		{9, 3},
		{10, 4},
		{99, 4},
		{100, 5},
		{999, 5},
		{1000, 6},
		{9999, 6},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := calcGutterWidth(tt.lineCount)
			if got != tt.want {
				t.Errorf("calcGutterWidth(%d) = %d, want %d", tt.lineCount, got, tt.want)
			}
		})
	}
}

func TestRenderBody_Dispatch(t *testing.T) {
	r := NewRenderer(80)

	tests := []struct {
		name      string
		blockType BlockType
		body      string
		wantErr   bool
	}{
		{"Execute transcript", BlockTypeExecute, "output", false},
		{"Notice transcript", BlockTypeNotice, "notice", false},
		{"Read code", BlockTypeRead, "package main", false},
		{"Grep code", BlockTypeGrep, "match", false},
		{"Patch diff", BlockTypeApplyPatch, "@@ diff", false},
		{"Plan list", BlockTypePlan, "• item", false},
		{"Summary list", BlockTypeSummary, "• item", false},
		{"Testing list", BlockTypeTesting, "• test", false},
		{"Error special", BlockTypeError, "error", false},
		{"Empty body", BlockTypeExecute, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewBlock(tt.blockType)
			block.Body = tt.body

			_, err := r.RenderBody(block)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderTranscript(t *testing.T) {
	r := NewRenderer(80)
	block := NewBlock(BlockTypeExecute)
	block.Body = "line 1\nline 2\nline 3"

	output, err := r.renderTranscript(block)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	if !strings.Contains(output, "line 1") {
		t.Error("Output should contain line 1")
	}
}

func TestRenderList_BulletTypes(t *testing.T) {
	r := NewRenderer(80)
	block := NewBlock(BlockTypePlan)

	tests := []struct {
		name  string
		body  string
		want  string // Substring that should be present
	}{
		{"Pending bullet", "• Pending task", "•"},
		{"Done checkmark", "✓ Done task", "✓"},
		{"Skipped hollow", "◦ Skipped task", "◦"},
		{"Markdown pending", "- [ ] Pending", "•"},
		{"Markdown done", "- [x] Done", "✓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block.Body = tt.body
			output, err := r.renderList(block)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(output, tt.want) {
				t.Errorf("renderList(%q) output should contain %q, got: %s", tt.body, tt.want, output)
			}
		})
	}
}

func TestRenderDiff_LineColors(t *testing.T) {
	r := NewRenderer(80)
	block := NewBlock(BlockTypeApplyPatch)
	block.Body = "@@ -1,3 +1,4 @@\n context line\n-removed line\n+added line"

	output, err := r.renderDiff(block)
	if err != nil {
		t.Fatal(err)
	}

	// Check that diff markers are present
	if !strings.Contains(output, "@@") {
		t.Error("Output should contain hunk header")
	}
	if !strings.Contains(output, "-removed line") {
		t.Error("Output should contain removed line")
	}
	if !strings.Contains(output, "+added line") {
		t.Error("Output should contain added line")
	}
}

func TestRenderCode_LineNumbers(t *testing.T) {
	r := NewRenderer(80)
	block := NewBlock(BlockTypeRead)
	block.Body = "line 1\nline 2\nline 3\nline 4\nline 5"

	output, err := r.renderCode(block)
	if err != nil {
		t.Fatal(err)
	}

	// Check that line numbers are present
	if !strings.Contains(output, "│") {
		t.Error("Output should contain gutter separator")
	}

	// Each line should be numbered
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}

func TestRenderError_FirstLineBold(t *testing.T) {
	r := NewRenderer(80)
	block := NewBlock(BlockTypeError)
	block.Body = "Error: Something went wrong\nStack trace line 1\nStack trace line 2"

	output, err := r.renderError(block)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "Error: Something went wrong") {
		t.Error("Output should contain error message")
	}
	if !strings.Contains(output, "Stack trace") {
		t.Error("Output should contain stack trace")
	}
}
