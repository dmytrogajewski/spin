// Package main demonstrates theme support in the TUI.
//
// Usage:
//   # Dark theme (default)
//   go run examples/theme-demo/main.go
//
//   # Light theme
//   SPIN_THEME=light go run examples/theme-demo/main.go
//
//   # 8-color fallback
//   TERM=xterm go run examples/theme-demo/main.go
package main

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/theme"
)

func main() {
	// Get theme from environment
	th := theme.GetThemeFromEnv()

	fmt.Println("Theme Demo")
	fmt.Println("==========")
	fmt.Println()

	// Show theme colors
	fmt.Printf("Theme colors:\n")
	fmt.Printf("  Fg:      %sforeground%s\n", th.Fg(), th.Reset())
	fmt.Printf("  Muted:   %smuted text%s\n", th.Muted(), th.Reset())
	fmt.Printf("  Blue:    %sblue%s\n", th.Blue(), th.Reset())
	fmt.Printf("  Green:   %sgreen%s\n", th.Green(), th.Reset())
	fmt.Printf("  Yellow:  %syellow%s\n", th.Yellow(), th.Reset())
	fmt.Printf("  Red:     %sred%s\n", th.Red(), th.Reset())
	fmt.Printf("  Magenta: %smagenta%s\n", th.Magenta(), th.Reset())
	fmt.Printf("  Cyan:    %scyan%s\n", th.Cyan(), th.Reset())
	fmt.Println()

	// Create a block renderer with theme
	renderer := blocks.NewRendererWithTheme(80, th)

	// Create sample blocks
	block1 := &blocks.Block{
		ID:        "demo1",
		Type:      blocks.BlockTypeExecute,
		Title:     "Example command",
		Body:      "$ go test ./...\nok    github.com/example/pkg 0.123s\n",
		FoldState: blocks.FoldStateExpanded,
		Severity:  blocks.SeverityInfo,
		Meta: map[string]interface{}{
			"command":  "go test ./...",
			"cwd":      "/home/user/project",
			"exitCode": 0,
		},
	}

	block2 := &blocks.Block{
		ID:        "demo2",
		Type:      blocks.BlockTypePlan,
		Title:     "Updated: 3 total (0 pending, 0 in progress, 3 completed)",
		Body:      "✓ Read spec\n✓ Implement feature\n✓ Write tests\n",
		FoldState: blocks.FoldStateExpanded,
		Severity:  blocks.SeverityInfo,
	}

	block3 := &blocks.Block{
		ID:        "demo3",
		Type:      blocks.BlockTypeError,
		Title:     "Error occurred",
		Body:      "panic: runtime error: index out of range\n",
		FoldState: blocks.FoldStateExpanded,
		Severity:  blocks.SeverityError,
	}

	fmt.Println("Sample blocks rendered with theme:")
	fmt.Println("-----------------------------------")
	fmt.Println()

	output1, _ := renderer.Render(block1)
	fmt.Print(output1)

	fmt.Println()

	output2, _ := renderer.Render(block2)
	fmt.Print(output2)

	fmt.Println()

	output3, _ := renderer.Render(block3)
	fmt.Print(output3)

	fmt.Println("\nTheme capabilities:")
	cap := theme.DetectTerminalCapabilities()
	switch cap {
	case theme.TerminalCapability8Color:
		fmt.Println("  8-color terminal detected")
	case theme.TerminalCapability256Color:
		fmt.Println("  256-color terminal detected")
	case theme.TerminalCapabilityTrueColor:
		fmt.Println("  True-color terminal detected")
	}

	fmt.Println("\nSet SPIN_THEME environment variable to 'dark' or 'light' to change theme.")
}
