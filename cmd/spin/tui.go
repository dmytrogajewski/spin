package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/manager"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/spf13/cobra"
)

// newTUICmd creates the TUI command for interactive terminal mode.
func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal UI mode",
		Long: `Launch Spin with an interactive terminal user interface.

The TUI provides a native-scrollback interface with:
  • Block-based timeline (EXECUTE, READ, APPLY_PATCH, etc.)
  • Real-time LLM streaming
  • Keyboard-first navigation (PgUp/PgDn, g/G)
  • Command palette (Ctrl-P)
  • Timeline filtering (/)
  • Approval dialogs for dangerous commands

Examples:
  spin tui
  spin tui --model llama3.1
  spin tui --provider anthropic --model claude-3-5-sonnet-20241022`,
		RunE: runTUI,
	}

	// TUI-specific flags
	cmd.Flags().Int("max-turns", 50, "Maximum conversation turns")

	return cmd
}

// runTUI executes the TUI mode.
func runTUI(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	configLoader, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	authMgr := createAuthManager()
	provider, err := buildProvider(ctx, configLoader, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}

	// Set max tokens for context percentage display
	// Use the provider's context window size
	ui.SetMaxTokens(128000) // Default for modern models, will be updated if provider reports different

	mgr, err := createManagerForTUI(provider, maxTurns, configLoader, ui)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Start UI in background
	uiCtx, uiCancel := context.WithCancel(ctx)
	defer uiCancel()

	go func() {
		if err := ui.Run(uiCtx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		}
	}()
	defer ui.Stop()

	workDir := getWorkingDirectory()
	conv, err := mgr.NewConversation(ctx, workDir)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer mgr.Close()

	// Initialize UI with conversation metadata
	initializeUI(ui, conv, provider)

	// Create event mapper
	mapper := tui.NewTUIMapper(ui)
	defer mapper.Close()

	// Start streaming channel
	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})
	go func() {
		ui.PrintChunks(ctx, streamCh)
		close(streamDone)
	}()

	// Print welcome message
	ui.PrintLine("")

	logo := "\x1b[49m                   \x1b[38;2;98;139;57;49m▄\x1b[38;2;169;237;83;49m▄\x1b[38;2;182;243;98;49m▄\x1b[38;2;186;243;101;49m▄\x1b[38;2;184;242;102;49m▄\x1b[38;2;186;240;103;49m▄\x1b[38;2;177;236;89;49m▄\x1b[49m                    \x1b[m\n" +
		"\x1b[49m               \x1b[38;2;182;239;102;49m▄\x1b[38;2;182;237;100;49m▄\x1b[38;2;182;239;100;48;2;180;239;98m▄\x1b[38;2;182;239;100;48;2;183;239;99m▄\x1b[38;2;184;238;100;48;2;184;239;100m▄\x1b[38;2;184;240;104;48;2;185;237;100m▄\x1b[38;2;176;232;89;48;2;182;237;99m▄\x1b[38;2;170;204;110;48;2;182;238;99m▄\x1b[38;2;155;190;118;48;2;186;238;101m▄\x1b[38;2;155;179;101;48;2;183;240;99m▄\x1b[38;2;173;207;114;48;2;182;238;100m▄\x1b[38;2;177;238;94;48;2;185;240;101m▄\x1b[38;2;184;239;102;48;2;184;239;100m▄\x1b[38;2;184;239;101;48;2;181;239;102m▄\x1b[38;2;185;238;101;49m▄\x1b[38;2;179;235;99;49m▄\x1b[49m               \x1b[m\n" +
		"\x1b[49m             \x1b[38;2;181;236;98;49m▄\x1b[38;2;183;240;101;48;2;185;239;100m▄\x1b[38;2;184;239;99;48;2;183;238;100m▄\x1b[38;2;184;239;99;48;2;179;235;96m▄\x1b[49;38;2;181;240;98m▀\x1b[49;38;2;180;237;90m▀\x1b[49m          \x1b[49;38;2;187;245;98m▀\x1b[49;38;2;183;238;102m▀\x1b[38;2;187;241;104;48;2;180;235;99m▄\x1b[38;2;182;238;97;49m▄\x1b[49m             \x1b[m\n" +
		"\x1b[49m           \x1b[38;2;179;235;114;49m▄\x1b[38;2;182;240;99;48;2;182;237;100m▄\x1b[38;2;182;239;98;48;2;182;239;99m▄\x1b[38;2;175;241;101;48;2;183;240;101m▄\x1b[49;38;2;183;240;92m▀\x1b[49m     \x1b[38;2;181;238;97;49m▄\x1b[38;2;182;237;99;49m▄\x1b[38;2;180;239;95;49m▄\x1b[38;2;185;240;97;49m▄\x1b[38;2;182;241;104;49m▄\x1b[38;2;183;240;87;49m▄\x1b[49m     \x1b[49;38;2;180;237;96m▀\x1b[38;2;185;237;105;48;2;182;237;101m▄\x1b[38;2;181;239;96;49m▄\x1b[49m           \x1b[m\n" +
		"\x1b[49m          \x1b[38;2;186;238;102;49m▄\x1b[38;2;182;235;98;48;2;180;236;98m▄\x1b[38;2;183;237;100;48;2;184;238;100m▄\x1b[49;38;2;184;241;99m▀\x1b[49m    \x1b[38;2;179;239;100;49m▄\x1b[38;2;183;238;101;48;2;183;237;98m▄\x1b[38;2;182;238;100;48;2;184;240;101m▄\x1b[38;2;184;237;100;48;2;183;238;100m▄\x1b[38;2;183;237;99;48;2;185;240;102m▄\x1b[38;2;182;237;97;48;2;181;237;97m▄\x1b[38;2;183;241;103;48;2;183;241;100m▄\x1b[38;2;185;241;103;48;2;184;239;101m▄\x1b[38;2;183;238;99;48;2;182;238;99m▄\x1b[38;2;182;238;99;48;2;182;238;98m▄\x1b[38;2;182;238;99;48;2;183;237;102m▄\x1b[38;2;184;239;102;49m▄\x1b[49m    \x1b[49;38;2;182;241;99m▀\x1b[38;2;181;239;100;48;2;197;221;145m▄\x1b[49m          \x1b[m\n" +
		"\x1b[49m         \x1b[38;2;223;252;141;49m▄\x1b[38;2;184;238;100;48;2;184;238;99m▄\x1b[38;2;183;239;99;48;2;185;242;101m▄\x1b[38;2;202;234;137;48;2;180;240;96m▄\x1b[49m   \x1b[38;2;227;245;166;49m▄\x1b[38;2;182;236;98;48;2;186;239;104m▄\x1b[38;2;185;240;102;48;2;182;238;99m▄\x1b[38;2;181;239;98;48;2;185;239;100m▄\x1b[49;38;2;184;240;100m▀\x1b[49m      \x1b[49;38;2;184;240;101m▀\x1b[38;2;203;237;123;48;2;183;237;100m▄\x1b[38;2;181;235;96;48;2;186;240;102m▄\x1b[38;2;181;238;98;48;2;180;237;97m▄\x1b[38;2;181;235;98;49m▄\x1b[49m   \x1b[49;38;2;180;238;99m▀\x1b[38;2;190;243;103;49m▄\x1b[49m         \x1b[m\n" +
		"\x1b[49m         \x1b[38;2;183;239;104;48;2;181;239;93m▄\x1b[38;2;182;238;98;48;2;183;239;100m▄\x1b[38;2;183;237;100;48;2;184;238;101m▄\x1b[49m    \x1b[38;2;184;239;98;48;2;184;241;97m▄\x1b[38;2;186;240;102;48;2;184;239;101m▄\x1b[38;2;178;236;88;48;2;182;239;98m▄\x1b[49m    \x1b[38;2;232;255;159;49m▄\x1b[38;2;106;137;55;49m▄\x1b[49m    \x1b[49;38;2;213;247;130m▀\x1b[38;2;184;240;100;48;2;182;238;99m▄\x1b[38;2;186;241;103;48;2;181;238;99m▄\x1b[38;2;181;236;98;48;2;179;235;93m▄\x1b[49m   \x1b[38;2;240;255;170;48;2;182;238;105m▄\x1b[49m         \x1b[m\n" +
		"\x1b[49m         \x1b[38;2;183;240;102;48;2;183;239;98m▄\x1b[38;2;184;238;100;48;2;180;235;97m▄\x1b[38;2;186;242;102;48;2;181;236;99m▄\x1b[49m    \x1b[38;2;185;238;100;48;2;184;238;99m▄\x1b[38;2;183;239;100;48;2;181;237;98m▄\x1b[38;2;172;203;117;48;2;163;192;116m▄\x1b[49m   \x1b[38;2;182;238;99;48;2;186;243;101m▄\x1b[38;2;182;238;102;48;2;179;237;98m▄\x1b[38;2;180;238;100;48;2;181;239;99m▄\x1b[38;2;183;239;101;48;2;184;239;102m▄\x1b[38;2;181;236;97;49m▄\x1b[49m   \x1b[49;38;2;190;217;142m▀\x1b[38;2;184;240;102;48;2;181;236;98m▄\x1b[38;2;185;241;102;48;2;183;237;98m▄\x1b[38;2;179;238;100;48;2;193;217;124m▄\x1b[49m            \x1b[m\n" +
		"\x1b[49m         \x1b[38;2;182;237;103;48;2;178;235;95m▄\x1b[38;2;184;239;100;48;2;182;237;99m▄\x1b[38;2;185;239;102;48;2;183;240;98m▄\x1b[49m    \x1b[38;2;184;237;95;48;2;181;238;97m▄\x1b[38;2;182;238;99;48;2;183;237;100m▄\x1b[38;2;182;239;99;48;2;176;238;94m▄\x1b[49m      \x1b[38;2;183;238;101;48;2;183;239;103m▄\x1b[38;2;186;242;102;48;2;179;235;97m▄\x1b[38;2;199;245;119;48;2;205;236;148m▄\x1b[49m   \x1b[38;2;182;238;95;48;2;184;239;99m▄\x1b[38;2;184;238;101;48;2;182;236;99m▄\x1b[38;2;181;238;96;48;2;182;239;100m▄\x1b[49m            \x1b[m\n" +
		"\x1b[49m         \x1b[49;38;2;200;246;121m▀\x1b[38;2;186;240;104;48;2;183;237;100m▄\x1b[38;2;183;239;99;48;2;183;238;100m▄\x1b[38;2;183;242;102;48;2;179;205;110m▄\x1b[49m   \x1b[49;38;2;217;237;169m▀\x1b[38;2;184;241;102;48;2;185;239;100m▄\x1b[38;2;182;239;100;48;2;183;239;100m▄\x1b[38;2;183;238;100;48;2;180;238;97m▄\x1b[38;2;179;236;96;49m▄\x1b[38;2;191;243;107;49m▄\x1b[49m \x1b[38;2;210;247;131;49m▄\x1b[38;2;182;239;100;49m▄\x1b[38;2;179;238;98;48;2;184;238;100m▄\x1b[38;2;182;240;99;48;2;181;237;99m▄\x1b[49;38;2;129;168;86m▀\x1b[49m   \x1b[38;2;185;239;103;48;2;183;239;98m▄\x1b[38;2;187;241;105;48;2;184;240;101m▄\x1b[38;2;181;241;104;48;2;181;238;99m▄\x1b[49m            \x1b[m\n" +
		"\x1b[49m          \x1b[38;2;173;198;123;48;2;183;239;102m▄\x1b[38;2;183;239;101;48;2;183;238;100m▄\x1b[38;2;183;238;99;48;2;183;238;101m▄\x1b[38;2;184;240;98;49m▄\x1b[49m    \x1b[49;38;2;186;239;103m▀\x1b[38;2;199;229;144;48;2;181;237;100m▄\x1b[38;2;185;238;100;48;2;183;238;100m▄\x1b[38;2;181;236;98;48;2;183;238;100m▄\x1b[38;2;184;239;100;48;2;185;238;100m▄\x1b[38;2;182;237;98;48;2;182;238;100m▄\x1b[38;2;179;236;98;48;2;181;237;99m▄\x1b[49;38;2;182;237;99m▀\x1b[49m    \x1b[38;2;181;237;99;48;2;222;238;163m▄\x1b[38;2;183;238;101;48;2;184;238;100m▄\x1b[38;2;185;241;105;48;2;184;238;100m▄\x1b[49;38;2;183;206;123m▀\x1b[49m            \x1b[m\n" +
		"\x1b[49m           \x1b[49;38;2;182;238;102m▀\x1b[38;2;182;238;100;48;2;184;240;100m▄\x1b[38;2;183;237;99;48;2;181;236;100m▄\x1b[38;2;182;239;101;48;2;178;238;105m▄\x1b[38;2;181;239;100;49m▄\x1b[49m            \x1b[38;2;181;238;94;49m▄\x1b[38;2;183;239;100;48;2;179;241;86m▄\x1b[38;2;185;238;99;48;2;186;240;103m▄\x1b[38;2;184;238;101;48;2;186;241;104m▄\x1b[49;38;2;183;241;91m▀\x1b[49m             \x1b[m\n" +
		"\x1b[49m             \x1b[49;38;2;182;237;98m▀\x1b[38;2;184;235;99;48;2;181;235;99m▄\x1b[38;2;182;237;98;48;2;184;239;101m▄\x1b[38;2;182;237;98;48;2;185;242;102m▄\x1b[38;2;181;236;96;49m▄\x1b[38;2;184;239;102;49m▄\x1b[38;2;228;254;159;49m▄\x1b[49m    \x1b[38;2;98;128;49;49m▄\x1b[38;2;182;237;99;49m▄\x1b[38;2;180;236;98;49m▄\x1b[38;2;184;240;101;48;2;183;239;100m▄\x1b[38;2;182;238;98;48;2;183;238;99m▄\x1b[38;2;183;240;101;48;2;184;239;101m▄\x1b[38;2;169;205;125;48;2;184;239;100m▄\x1b[49;38;2;195;225;120m▀\x1b[49m              \x1b[m\n" +
		"\x1b[49m               \x1b[49;38;2;185;241;99m▀\x1b[49;38;2;184;238;100m▀\x1b[38;2;182;240;99;48;2;183;238;100m▄\x1b[38;2;182;238;101;48;2;183;238;98m▄\x1b[38;2;183;239;100;48;2;185;235;100m▄\x1b[38;2;184;239;100;48;2;186;240;103m▄\x1b[48;2;183;237;100m \x1b[38;2;183;238;101;48;2;183;239;99m▄\x1b[38;2;185;239;101;48;2;185;239;100m▄\x1b[38;2;184;239;100;48;2;179;235;97m▄\x1b[38;2;184;239;101;48;2;184;239;100m▄\x1b[38;2;181;238;98;48;2;184;239;100m▄\x1b[38;2;202;238;133;48;2;185;239;102m▄\x1b[49;38;2;181;237;96m▀\x1b[49m                 \x1b[m\n" +
		"\x1b[49m                   \x1b[49;38;2;79;111;31m▀\x1b[49;38;2;176;237;84m▀\x1b[49;38;2;187;242;91m▀\x1b[49;38;2;186;246;98m▀\x1b[49;38;2;176;236;87m▀\x1b[49;38;2;225;251;166m▀\x1b[49m                     \x1b[m\n" +
		"\x1b[49m                                              \x1b[m\n" +
		"\x1b[49m  \x1b[38;2;229;230;228;49m▄\x1b[38;2;248;248;238;49m▄\x1b[38;2;248;247;237;49m▄\x1b[38;2;250;249;239;49m▄\x1b[38;2;247;245;236;49m▄\x1b[38;2;249;248;239;49m▄\x1b[38;2;250;249;240;49m▄\x1b[38;2;249;249;240;49m▄\x1b[38;2;249;247;239;49m▄\x1b[38;2;254;253;248;49m▄\x1b[49m  \x1b[38;2;250;248;239;49m▄\x1b[38;2;249;248;239;49m▄\x1b[38;2;248;246;237;49m▄\x1b[38;2;249;247;238;49m▄\x1b[38;2;251;252;244;49m▄\x1b[38;2;249;247;239;49m▄\x1b[38;2;249;248;239;49m▄\x1b[38;2;250;248;240;49m▄\x1b[38;2;249;249;240;49m▄▄\x1b[49m   \x1b[38;2;252;250;249;49m▄\x1b[38;2;250;248;239;49m▄\x1b[38;2;248;247;239;49m▄\x1b[49m   \x1b[38;2;247;245;237;49m▄\x1b[38;2;247;245;236;49m▄\x1b[38;2;250;248;240;49m▄▄\x1b[49m     \x1b[38;2;250;248;239;49m▄\x1b[38;2;250;249;240;49m▄\x1b[38;2;253;253;252;49m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m \x1b[38;2;245;247;239;49m▄\x1b[38;2;249;247;238;48;2;247;247;235m▄\x1b[38;2;250;249;240;48;2;250;248;239m▄\x1b[38;2;248;246;238;48;2;248;247;239m▄\x1b[38;2;248;246;242;48;2;249;246;238m▄\x1b[38;2;250;248;243;48;2;250;248;239m▄\x1b[38;2;250;249;244;48;2;249;248;238m▄\x1b[38;2;247;247;242;48;2;246;245;235m▄\x1b[38;2;248;248;241;48;2;248;248;239m▄\x1b[38;2;248;245;240;48;2;247;246;238m▄\x1b[38;2;251;251;245;48;2;253;253;250m▄\x1b[49m  \x1b[38;2;249;248;239;48;2;249;246;237m▄\x1b[38;2;250;248;239;48;2;248;246;237m▄\x1b[38;2;248;247;238;48;2;249;247;237m▄\x1b[38;2;249;247;241;48;2;248;247;236m▄\x1b[38;2;246;245;240;48;2;250;249;240m▄\x1b[38;2;248;245;242;48;2;247;248;238m▄\x1b[38;2;248;246;240;48;2;249;248;240m▄\x1b[38;2;247;247;240;48;2;250;248;239m▄\x1b[38;2;250;248;239;48;2;248;247;238m▄\x1b[38;2;249;247;239;48;2;251;249;241m▄\x1b[38;2;249;248;239;48;2;250;249;240m▄\x1b[38;2;248;245;237;49m▄\x1b[49m \x1b[38;2;252;251;248;48;2;251;250;247m▄\x1b[38;2;247;245;237;48;2;248;247;238m▄\x1b[38;2;248;246;238;48;2;250;248;239m▄\x1b[49m   \x1b[38;2;249;248;239;48;2;248;246;238m▄\x1b[38;2;249;248;239;48;2;251;249;241m▄\x1b[38;2;248;247;238;48;2;249;248;238m▄\x1b[38;2;249;247;238;48;2;249;248;239m▄\x1b[38;2;246;246;237;48;2;248;246;235m▄\x1b[49m    \x1b[38;2;250;248;240;48;2;250;248;239m▄\x1b[38;2;248;247;238;48;2;252;249;241m▄\x1b[38;2;254;254;250;48;2;254;254;253m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m \x1b[38;2;250;247;238;48;2;248;246;237m▄\x1b[38;2;250;249;241;48;2;249;247;239m▄\x1b[38;2;247;246;236;48;2;250;248;240m▄\x1b[49m          \x1b[38;2;249;249;240;48;2;248;246;237m▄\x1b[38;2;247;246;236;48;2;249;248;238m▄\x1b[38;2;248;248;238;48;2;249;248;240m▄\x1b[49m      \x1b[38;2;249;247;239;48;2;248;247;238m▄\x1b[38;2;251;249;240;48;2;248;247;236m▄\x1b[38;2;247;246;237;48;2;249;247;237m▄\x1b[49m \x1b[38;2;253;253;249;48;2;252;252;248m▄\x1b[38;2;248;246;237;48;2;248;248;238m▄\x1b[38;2;248;247;239;48;2;250;249;240m▄\x1b[49m   \x1b[38;2;248;246;236;48;2;248;247;239m▄\x1b[38;2;251;248;240;48;2;248;245;237m▄\x1b[38;2;245;246;235;48;2;249;249;240m▄\x1b[38;2;246;247;238;48;2;250;249;240m▄\x1b[38;2;248;245;238;48;2;251;250;241m▄\x1b[38;2;248;248;237;48;2;248;247;239m▄\x1b[38;2;221;221;221;49m▄\x1b[49m  \x1b[38;2;247;246;236;48;2;247;247;238m▄\x1b[48;2;248;246;237m \x1b[38;2;254;254;252;48;2;254;253;252m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m \x1b[49;38;2;253;254;251m▀\x1b[38;2;248;246;239;48;2;247;246;237m▄\x1b[38;2;249;247;239;48;2;250;248;237m▄\x1b[38;2;250;248;240;48;2;249;247;238m▄\x1b[38;2;246;244;236;48;2;249;248;239m▄\x1b[38;2;250;248;240;48;2;247;246;236m▄\x1b[38;2;248;246;238;48;2;249;248;239m▄\x1b[38;2;249;248;239;48;2;247;246;237m▄\x1b[38;2;247;246;237;48;2;248;246;238m▄\x1b[38;2;249;247;238;48;2;248;247;238m▄\x1b[38;2;250;248;238;49m▄\x1b[49m  \x1b[38;2;247;245;238;48;2;248;248;238m▄\x1b[38;2;249;248;239;48;2;250;248;240m▄\x1b[38;2;248;248;238;48;2;248;246;238m▄\x1b[38;2;248;246;238;49m▄\x1b[38;2;249;247;238;49m▄\x1b[38;2;248;248;239;49m▄\x1b[38;2;250;249;241;49m▄\x1b[38;2;249;247;239;49m▄\x1b[38;2;248;247;238;49m▄\x1b[38;2;248;248;239;48;2;247;246;236m▄\x1b[38;2;248;246;237;48;2;249;247;238m▄\x1b[38;2;252;250;249;48;2;249;247;238m▄\x1b[49m \x1b[48;2;251;251;248m \x1b[38;2;248;247;238;48;2;249;247;238m▄\x1b[38;2;249;248;239;48;2;250;248;240m▄\x1b[49m   \x1b[38;2;248;246;239;48;2;249;247;239m▄\x1b[38;2;248;247;238;48;2;249;247;239m▄\x1b[38;2;247;246;239;48;2;248;247;238m▄\x1b[49m \x1b[38;2;249;247;241;48;2;247;246;237m▄\x1b[38;2;249;247;238;48;2;247;246;237m▄\x1b[38;2;249;246;237;48;2;247;245;238m▄\x1b[38;2;248;248;245;49m▄\x1b[49m \x1b[38;2;249;247;238;48;2;249;247;239m▄\x1b[38;2;250;249;240;48;2;249;247;238m▄\x1b[38;2;254;253;252;48;2;253;253;252m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m   \x1b[49;38;2;253;254;252m▀\x1b[49;38;2;247;247;239m▀\x1b[49;38;2;247;246;237m▀\x1b[49;38;2;251;249;239m▀\x1b[49;38;2;250;248;238m▀\x1b[49;38;2;248;246;238m▀\x1b[49;38;2;249;246;238m▀\x1b[38;2;250;248;241;48;2;247;246;238m▄\x1b[38;2;248;246;238;48;2;250;249;241m▄\x1b[38;2;210;210;209;49m▄\x1b[49m \x1b[38;2;247;246;237;48;2;248;246;237m▄\x1b[38;2;249;247;238;48;2;251;249;241m▄\x1b[38;2;249;248;239;48;2;251;251;241m▄\x1b[38;2;249;248;238;48;2;250;248;240m▄\x1b[38;2;248;247;237;48;2;250;248;240m▄\x1b[38;2;249;247;239;48;2;249;248;240m▄\x1b[38;2;251;249;240;48;2;250;248;239m▄\x1b[38;2;249;247;239;48;2;248;247;238m▄\x1b[38;2;250;248;239;48;2;248;248;238m▄\x1b[38;2;248;248;236;48;2;250;248;240m▄\x1b[49;38;2;249;250;238m▀\x1b[49m  \x1b[38;2;252;251;248;48;2;252;252;248m▄\x1b[48;2;248;246;238m \x1b[38;2;250;248;239;48;2;250;247;240m▄\x1b[49m   \x1b[38;2;247;245;237;48;2;249;246;238m▄\x1b[38;2;250;249;240;48;2;250;248;238m▄\x1b[38;2;245;244;234;48;2;245;246;237m▄\x1b[49m  \x1b[38;2;242;242;240;48;2;249;248;238m▄\x1b[38;2;249;248;239;48;2;248;246;239m▄\x1b[38;2;249;248;239;48;2;249;246;238m▄\x1b[38;2;249;247;239;49m▄\x1b[38;2;247;247;238;48;2;250;249;240m▄\x1b[38;2;249;247;238;48;2;247;246;238m▄\x1b[38;2;253;254;251;48;2;254;252;252m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m \x1b[38;2;239;239;229;49m▄\x1b[38;2;248;247;238;49m▄\x1b[38;2;247;246;237;49m▄\x1b[38;2;249;248;238;49m▄▄\x1b[38;2;250;250;240;49m▄\x1b[38;2;247;246;236;49m▄\x1b[38;2;248;246;239;49m▄\x1b[38;2;249;248;239;48;2;226;224;220m▄\x1b[38;2;249;248;239;48;2;248;247;238m▄\x1b[38;2;248;246;237;48;2;248;247;238m▄\x1b[49;38;2;254;254;254m▀\x1b[49m \x1b[38;2;249;248;238;48;2;248;247;238m▄\x1b[38;2;247;245;236;48;2;249;248;239m▄\x1b[38;2;248;248;239;48;2;249;248;238m▄\x1b[49m          \x1b[38;2;251;251;248;48;2;252;252;247m▄\x1b[38;2;249;248;240;48;2;249;247;238m▄\x1b[38;2;248;246;238;48;2;250;248;239m▄\x1b[49m   \x1b[38;2;247;246;238;48;2;248;247;239m▄\x1b[38;2;249;247;237;48;2;249;247;239m▄\x1b[38;2;247;246;238;48;2;245;243;234m▄\x1b[49m   \x1b[49;38;2;248;247;237m▀\x1b[38;2;248;244;236;48;2;250;248;239m▄\x1b[38;2;251;248;239;48;2;248;246;237m▄\x1b[38;2;250;247;238;48;2;249;247;238m▄\x1b[38;2;249;248;240;48;2;248;247;238m▄\x1b[38;2;253;253;251;48;2;254;254;253m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m \x1b[38;2;233;233;223;48;2;242;241;233m▄\x1b[38;2;244;243;235;48;2;247;245;237m▄\x1b[38;2;243;242;233;48;2;249;247;239m▄\x1b[38;2;248;245;237;48;2;250;248;239m▄\x1b[38;2;244;243;235;48;2;247;246;237m▄\x1b[38;2;244;243;236;48;2;249;247;238m▄\x1b[38;2;241;239;229;48;2;249;248;239m▄\x1b[38;2;245;242;235;48;2;247;247;238m▄\x1b[38;2;249;249;243;48;2;248;247;238m▄\x1b[49;38;2;248;247;238m▀\x1b[49;38;2;249;248;240m▀\x1b[49m  \x1b[38;2;250;248;241;48;2;248;247;238m▄\x1b[38;2;250;248;242;48;2;249;247;238m▄\x1b[38;2;247;247;239;48;2;250;249;242m▄\x1b[49m          \x1b[38;2;250;250;246;48;2;251;251;249m▄\x1b[38;2;243;242;233;48;2;249;248;239m▄\x1b[38;2;245;245;236;48;2;250;248;239m▄\x1b[49m   \x1b[38;2;245;244;235;48;2;248;246;238m▄\x1b[38;2;243;243;235;48;2;249;248;239m▄\x1b[38;2;241;241;233;48;2;242;243;233m▄\x1b[49m    \x1b[49;38;2;249;246;238m▀\x1b[38;2;245;245;236;48;2;249;247;238m▄\x1b[38;2;247;246;238;48;2;249;247;238m▄\x1b[38;2;245;245;238;48;2;248;246;237m▄\x1b[38;2;253;253;252;48;2;253;254;252m▄\x1b[49m \x1b[m\n" +
		"\x1b[49m                                              \x1b[m\n"
	ui.PrintLine(logo)
	ui.PrintLine("Type your prompt and press Enter.")
	ui.PrintLine("Commands: /mode [name], /help, /exit (or press Ctrl-D)\n")

	// Subscribe to conversation events
	eventStream := conv.Stream()

	// Start event processing loop
	eventDone := make(chan struct{})
	go func() {
		defer close(eventDone)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}
				if err := mapper.MapEvent(event); err != nil {
					ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", err))
				}

				// Update token count from conversation history after each event
				// This ensures the status bar always shows current cumulative total
				if event.Type == events.EventTurnComplete || event.Type == events.EventContentComplete {
					tokenCount := int64(conv.GetTokenCount())
					ui.SetTokenCount(tokenCount)
				}
			}
		}
	}()

	// Main input loop
	inputCh := ui.RequestInput()
	for {
		select {
		case <-ctx.Done():
			// Wait for event processing to finish
			<-eventDone
			return ctx.Err()

		case line, ok := <-inputCh:
			if !ok {
				// UI closed (Ctrl-D)
				<-eventDone
				return nil
			}

			if line == "" {
				continue
			}

			// Check if input is a command
			cmdResult := parseCommand(line)

			if cmdResult.isCommand {
				// Handle command
				_, err := handleCommand(ui, conv, cmdResult)
				if err != nil {
					if err.Error() == "exit requested" {
						<-eventDone
						return nil
					}
					ui.PrintLine(fmt.Sprintf("Command error: %v\n", err))
				}
				// Skip conversation turn for commands
				continue
			}

			// Submit prompt to conversation
			turnCtx, turnCancel := context.WithCancel(ctx)
			defer turnCancel()

			// Send message and handle errors
			err := conv.RunTurn(turnCtx, line)

			// Stop streaming to close the channel (this triggers final newline in PrintChunks)
			mapper.StopStreaming()

			// Wait for streaming to complete
			<-streamDone

			// Reset streamDone for next turn
			streamDone = make(chan struct{})
			streamCh = mapper.StartStreaming()
			go func() {
				ui.PrintChunks(ctx, streamCh)
				close(streamDone)
			}()

			if err != nil {
				ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
			}
		}
	}
}

// createManagerForTUI creates a core.Manager configured for TUI mode.
func createManagerForTUI(provider llm.Provider, maxTurns int, configLoader *config.Loader, ui *adapters.PureTTY) (*manager.Manager, error) {
	workDir := getWorkingDirectory()
	cfg := buildConfig(configLoader, maxTurns, workDir)

	// Create tool registry with simple tools (no dependencies)
	registry := tools.NewRegistry()

	// Register simple built-in tools (file I/O)
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewListDirectoryTool())

	// Note: ExecuteCommandTool and GetContextTool are registered by Agent
	// as they require executor, validator, and context dependencies

	// Create approval handler (always enabled)
	approvalHandler := func(req security.ApprovalRequest) security.ApprovalResponse {
		return ui.ShowApprovalDialog(req)
	}

	// Create manager with options
	var opts []manager.ManagerOption
	opts = append(opts, manager.WithLLM(provider))
	opts = append(opts, manager.WithManagerToolRegistry(registry))
	opts = append(opts, manager.WithManagerApprovalHandler(approvalHandler))

	mgr, err := manager.NewManager(cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}

	return mgr, nil
}

// setupSignalHandling sets up signal handling for graceful shutdown.
func setupSignalHandling(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()
}

// getWorkingDirectory returns the working directory for the conversation.
func getWorkingDirectory() string {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if flagWorkDir != "" {
		workDir = flagWorkDir
	}
	return workDir
}

// initializeUI initializes the UI with conversation metadata.
func initializeUI(ui *adapters.PureTTY, conv *conversation.Conversation, provider llm.Provider) {
	taskMode := conv.GetTaskMode()
	ui.SetTaskMode(taskMode)

	providerName := provider.Name()
	modelName := flagModel
	ui.SetProviderInfo(providerName, modelName)

	tokenCount := int64(conv.GetTokenCount())
	ui.SetTokenCount(tokenCount)

	sessionID := conv.GetSessionID()
	ui.SetConversationID(sessionID)
}

// buildConfig builds the configuration from multiple sources.
func buildConfig(configLoader *config.Loader, maxTurns int, workDir string) *manager.Config {
	cfg := manager.DefaultConfig()
	cfg.WorkDir = workDir

	// Layer 1: Load from config file
	var fileCfg manager.Config
	if err := configLoader.Unmarshal(&fileCfg); err == nil {
		applyFileConfig(cfg, &fileCfg)
	}

	// Layer 2: Override with CLI flags
	applyCLIFlags(cfg, maxTurns)

	return cfg
}

// applyFileConfig applies configuration from file to the main config.
func applyFileConfig(cfg *manager.Config, fileCfg *manager.Config) {
	if fileCfg.Provider != "" {
		cfg.Provider = fileCfg.Provider
	}
	if fileCfg.Model != "" {
		cfg.Model = fileCfg.Model
	}
	if fileCfg.MaxTurns > 0 {
		cfg.MaxTurns = fileCfg.MaxTurns
	}
	if fileCfg.Timeout > 0 {
		cfg.Timeout = fileCfg.Timeout
	}
	if fileCfg.MaxTokens > 0 {
		cfg.MaxTokens = fileCfg.MaxTokens
	}
}

// applyCLIFlags applies CLI flags to the configuration.
func applyCLIFlags(cfg *manager.Config, maxTurns int) {
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	if flagProvider != "" {
		cfg.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
}
