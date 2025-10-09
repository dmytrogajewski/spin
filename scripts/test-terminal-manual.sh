#!/bin/bash
# Manual Terminal Functionality Smoke Test
# Phase 1.1: Terminal Control Infrastructure
#
# This script provides a manual verification checklist for terminal
# functionality that requires human interaction and observation.
#
# Usage: ./scripts/test-terminal-manual.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Spin Terminal Control Manual Test ===${NC}"
echo ""
echo "This script will test terminal control functionality."
echo "You will be asked to verify certain behaviors manually."
echo ""

# Test 1: Raw Mode Enter/Exit
echo -e "${YELLOW}Test 1: Raw Mode Enter/Exit${NC}"
echo "Expected: Terminal enters raw mode (no echo), then exits cleanly"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_raw_mode.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Entering raw mode...")
	if err := tty.Enter(); err != nil {
		fmt.Fprintf(os.Stderr, "Enter error: %v\n", err)
		os.Exit(1)
	}
	defer tty.Exit()

	fmt.Print("Type some characters (they should not echo): ")
	time.Sleep(3 * time.Second)
	fmt.Println("\nExiting raw mode...")
}
EOF

go run /tmp/test_raw_mode.go
rm /tmp/test_raw_mode.go

echo ""
echo -n "Did characters NOT echo while in raw mode? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 1 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 1 PASSED${NC}"
echo ""

# Test 2: Cursor Visibility
echo -e "${YELLOW}Test 2: Cursor Hide/Show${NC}"
echo "Expected: Cursor disappears, then reappears"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_cursor.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	fmt.Print(term.HideCursor)
	fmt.Println("Cursor should be HIDDEN now (observe for 2 seconds)")
	time.Sleep(2 * time.Second)

	fmt.Print(term.ShowCursor)
	fmt.Println("Cursor should be VISIBLE now")
	time.Sleep(1 * time.Second)
}
EOF

go run /tmp/test_cursor.go
rm /tmp/test_cursor.go

echo ""
echo -n "Did the cursor hide and then show? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 2 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 2 PASSED${NC}"
echo ""

# Test 3: Line Clear
echo -e "${YELLOW}Test 3: Clear Line${NC}"
echo "Expected: Line of text appears, then clears"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_clear.go <<'EOF'
package main

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	fmt.Print("This text will be cleared")
	time.Sleep(2 * time.Second)
	fmt.Print("\r" + term.ClearLine)
	fmt.Println("Text cleared! This is new text.")
}
EOF

go run /tmp/test_clear.go
rm /tmp/test_clear.go

echo ""
echo -n "Did the first line clear before new text appeared? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 3 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 3 PASSED${NC}"
echo ""

# Test 4: Cursor Positioning
echo -e "${YELLOW}Test 4: Cursor Positioning${NC}"
echo "Expected: 'X' appears at different columns"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_position.go <<'EOF'
package main

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	positions := []int{1, 10, 20, 40, 60, 80}
	for _, col := range positions {
		fmt.Print("\r" + term.ClearLine)
		fmt.Print(term.MoveCursorToCol(col) + "X")
		time.Sleep(300 * time.Millisecond)
	}
	fmt.Println()
}
EOF

go run /tmp/test_position.go
rm /tmp/test_position.go

echo ""
echo -n "Did 'X' move to different columns (1, 10, 20, 40, 60, 80)? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 4 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 4 PASSED${NC}"
echo ""

# Test 5: Terminal Size Detection
echo -e "${YELLOW}Test 5: Terminal Size Detection${NC}"
echo "Current terminal size will be displayed"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_size.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	w, h := tty.Size()
	fmt.Printf("Terminal size: %d columns × %d rows\n", w, h)
}
EOF

go run /tmp/test_size.go
rm /tmp/test_size.go

echo ""
echo -n "Does the reported size match your terminal size? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 5 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 5 PASSED${NC}"
echo ""

# Test 6: Resize Detection (requires manual resize)
echo -e "${YELLOW}Test 6: Resize Detection${NC}"
echo "A program will monitor terminal size for 10 seconds."
echo "RESIZE YOUR TERMINAL WINDOW during this time."
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_resize.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resizes := 0
	tty.OnResize(func(w, h int) {
		resizes++
		fmt.Printf("\rResize detected! New size: %d×%d (resize #%d)", w, h, resizes)
	})

	w, h := tty.Size()
	fmt.Printf("Initial size: %d×%d\n", w, h)
	fmt.Println("Resize your terminal now... (10 seconds)")

	// Keep program running to receive SIGWINCH
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-sigCh:
			// SIGWINCH received, updateSize is called internally
		case <-timeout:
			fmt.Printf("\n\nTotal resizes detected: %d\n", resizes)
			return
		}
	}
}
EOF

go run /tmp/test_resize.go
rm /tmp/test_resize.go

echo ""
echo -n "Were resize events detected when you resized the terminal? [y/n]: "
read -n 1 response
echo ""
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo -e "${RED}✗ Test 6 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 6 PASSED${NC}"
echo ""

# Test 7: Terminal State Restoration
echo -e "${YELLOW}Test 7: Terminal State Restoration${NC}"
echo "Expected: Terminal returns to normal state after test"
echo "Press any key to start..."
read -n 1 -s

cat > /tmp/test_restore.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func main() {
	tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := tty.Enter(); err != nil {
		fmt.Fprintf(os.Stderr, "Enter error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("In raw mode for 2 seconds...")
	time.Sleep(2 * time.Second)

	if err := tty.Exit(); err != nil {
		fmt.Fprintf(os.Stderr, "Exit error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Exited raw mode - terminal should work normally now")
}
EOF

go run /tmp/test_restore.go
rm /tmp/test_restore.go

echo ""
echo "Try typing now. Does echo work normally?"
echo -n "Enter 'yes' if terminal works normally: "
read response
if [[ ! "$response" == "yes" ]]; then
    echo -e "${RED}✗ Test 7 FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Test 7 PASSED${NC}"
echo ""

# Summary
echo -e "${GREEN}=== All Manual Tests PASSED ===${NC}"
echo ""
echo "Phase 1.1 Terminal Control Infrastructure verification complete."
echo "Date: $(date)"
echo ""
echo "Verified functionality:"
echo "  ✓ Raw mode enter/exit"
echo "  ✓ Cursor visibility control"
echo "  ✓ Line clearing"
echo "  ✓ Cursor positioning"
echo "  ✓ Terminal size detection"
echo "  ✓ Resize event handling"
echo "  ✓ Terminal state restoration"
echo ""
