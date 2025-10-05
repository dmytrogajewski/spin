# Spin Manual Test Plan

**Date:** 2025-10-05
**Version:** v0.1.0
**Test Environment:** Fedora Linux with Ollama 127.0.0.1:11434

---

## Test Models Available

```
qwen3:0.6b          - 522 MB   (fastest, basic testing)
qwen3:1.7b          - 1.4 GB   (fast, good for quick tests)
granite3.3:2b       - 1.5 GB   (fast, good balance)
llama3.2:3b         - 2.0 GB   (medium, reliable)
qwen2.5-coder:7b    - 4.7 GB   (coding-focused, slower)
```

**Recommended for testing:** `qwen3:1.7b` or `granite3.3:2b` (fast + capable)

---

## Test Categories

1. **Non-Interactive Tests** (automated/scripted)
2. **Interactive TUI Tests** (manual)
3. **Integration Tests** (real LLM)
4. **Edge Cases & Error Handling**

---

## Part 1: Non-Interactive Tests (Automated)

### 1.1 Configuration Tests

**Test config loading and validation:**

```bash
# Test 1: Show current config
./bin/spin config show

# Test 2: Validate config
./bin/spin config validate

# Test 3: Show config path
./bin/spin config path

# Test 4: Show config in JSON format
./bin/spin config show --format json | jq

# Test 5: Show config in YAML format
./bin/spin config show --format yaml
```

**Expected Results:**
- ✅ Config loads without errors
- ✅ Validation passes
- ✅ Path points to correct location
- ✅ JSON/YAML output is valid and parseable

---

### 1.2 MCP Management Tests

**Test MCP server configuration:**

```bash
# Test 1: List MCP servers (should be empty initially)
./bin/spin mcp list

# Test 2: Add filesystem MCP server
./bin/spin mcp add filesystem npx -- -y @modelcontextprotocol/server-filesystem /tmp

# Test 3: List servers again
./bin/spin mcp list

# Test 4: Get server details
./bin/spin mcp get filesystem

# Test 5: Get server in JSON format
./bin/spin mcp get filesystem --format json | jq

# Test 6: Remove server
./bin/spin mcp remove filesystem --yes

# Test 7: Verify removal
./bin/spin mcp list
```

**Expected Results:**
- ✅ Initially empty list
- ✅ Add succeeds, shows in list
- ✅ Get returns correct details
- ✅ JSON output valid
- ✅ Remove succeeds, list empty again

---

### 1.3 Debug Commands Tests

**Test debug event logging:**

```bash
# Test 1: Debug events with simple prompt (will use mock provider)
./bin/spin debug events "say hello" 2>&1 | head -20

# Test 2: Debug events with JSON output
./bin/spin debug events --format json "say hello" 2>&1 | head -10

# Test 3: Debug events with filter (tool events only)
./bin/spin debug events --filter tool "list files" 2>&1

# Test 4: Debug events with multiple filters
./bin/spin debug events --filter tool,stream "test" 2>&1

# Test 5: Platform check - sandbox (should fail on Linux)
./bin/spin debug sandbox ls 2>&1 | grep -i "only available"

# Test 6: Platform check - landlock (should work on Linux)
./bin/spin debug landlock ls 2>&1
```

**Expected Results:**
- ✅ Events logged to stderr in real-time
- ✅ JSON output is valid
- ✅ Filters work correctly
- ✅ Platform checks prevent wrong OS usage

---

### 1.4 Exec Mode Tests (Non-Interactive)

**Test headless execution:**

```bash
# Test 1: Simple task with fast model
./bin/spin exec --model qwen3:0.6b --provider ollama "say hello world"

# Test 2: Task with output to file
./bin/spin exec --model qwen3:1.7b "write hello world in Python" > /tmp/exec_output.txt
cat /tmp/exec_output.txt

# Test 3: Task with JSON output format
./bin/spin exec --model qwen3:1.7b --format json "list 3 colors" | jq

# Test 4: Task with auto-approve (dangerous - test carefully)
echo "print('test')" > /tmp/test.py
./bin/spin exec --model qwen3:1.7b --auto-approve "run the python file /tmp/test.py"

# Test 5: Task with timeout
timeout 10s ./bin/spin exec --model qwen3:1.7b "write a long essay"

# Test 6: Task from stdin
echo "what is 2+2" | ./bin/spin exec --model qwen3:0.6b

# Test 7: Test with different providers
./bin/spin exec --provider ollama --model qwen3:0.6b "test ollama"
```

**Expected Results:**
- ✅ Tasks execute without TUI
- ✅ Output streams to stdout
- ✅ JSON output is valid
- ✅ Auto-approve works (commands execute)
- ✅ Timeout stops execution
- ✅ Stdin works
- ✅ Provider selection works

---

### 1.5 Version & Help Tests

```bash
# Test 1: Version command
./bin/spin version

# Test 2: Version flag
./bin/spin --version

# Test 3: Help command
./bin/spin --help

# Test 4: Subcommand help
./bin/spin exec --help
./bin/spin config --help
./bin/spin mcp --help
./bin/spin debug --help

# Test 5: Shell completions
./bin/spin completion bash > /tmp/spin_completion.bash
source /tmp/spin_completion.bash
# Test: type "spin " and press Tab
```

**Expected Results:**
- ✅ Version displays correctly
- ✅ Help is comprehensive and accurate
- ✅ Subcommand help works
- ✅ Completions generate without errors

---

## Part 2: Interactive TUI Tests (Manual)

### 2.1 TUI Launch & Basic UI

**Test basic TUI functionality:**

```bash
# Launch TUI with fast model
./bin/spin --model qwen3:1.7b --provider ollama
```

**Manual Test Steps:**

1. **Launch & Display:**
   - ✅ TUI launches without errors
   - ✅ Status bar shows: model, sandbox mode, directory, tokens
   - ✅ Input field is visible and focused
   - ✅ Chat area is empty (no messages yet)
   - ✅ No visual glitches or rendering issues

2. **Window Resize:**
   - Resize terminal window (smaller, larger)
   - ✅ TUI adapts correctly
   - ✅ No text overflow or truncation issues
   - ✅ Status bar adjusts

3. **Color & Theme:**
   - Check colors render correctly
   - ✅ User messages distinct from assistant messages
   - ✅ Status bar colors appropriate
   - ✅ NO_COLOR environment respected (if set)

**Exit:** Press `Ctrl+D`

---

### 2.2 Chat Interface & Message Flow

**Test message sending and streaming:**

```bash
./bin/spin --model qwen3:1.7b
```

**Manual Test Steps:**

1. **Send Simple Message:**
   - Type: "Hello! Say hi back."
   - Press Enter
   - ✅ Message appears in chat as "User: Hello! Say hi back."
   - ✅ Assistant response streams in real-time (character by character)
   - ✅ Status bar shows token usage increasing
   - ✅ Input field clears after sending

2. **Multi-line Input:**
   - Type: "Write a\nhaiku about\ncode" (use Shift+Enter for newlines if supported, or just multi-word)
   - Press Enter
   - ✅ Multi-line input works
   - ✅ Response appears

3. **Markdown Rendering:**
   - Type: "Show me a Python hello world with markdown code block"
   - ✅ Code block is syntax highlighted
   - ✅ Markdown formatting (bold, italic) renders
   - ✅ Line breaks preserved

4. **Long Response:**
   - Type: "List 20 programming languages with descriptions"
   - ✅ Response scrolls in viewport
   - ✅ Can scroll up/down with PgUp/PgDn
   - ✅ Scroll indicator shows position

5. **Error Handling:**
   - Stop Ollama server: `systemctl stop ollama` (or `pkill ollama`)
   - Type: "test"
   - ✅ Error displayed in UI (not crash)
   - ✅ Error message is actionable
   - Restart Ollama: `systemctl start ollama`

**Exit:** Press `Ctrl+D`

---

### 2.3 File Picker (@-trigger)

**Test file search and insertion:**

```bash
./bin/spin --model qwen3:1.7b
```

**Manual Test Steps:**

1. **Trigger File Picker:**
   - Type: "@"
   - ✅ File picker modal appears
   - ✅ Shows files from current directory

2. **Fuzzy Search:**
   - Type: "config"
   - ✅ List filters to files matching "config"
   - ✅ Real-time filtering works

3. **Navigation:**
   - Use ↑/↓ arrows to navigate
   - ✅ Selection highlights correctly
   - ✅ Can navigate through list

4. **Selection:**
   - Press Enter or Tab
   - ✅ File path inserted into input
   - ✅ File picker closes
   - ✅ Can continue typing

5. **Cancel:**
   - Type "@" again
   - Press Esc
   - ✅ File picker closes without selection

6. **Use Selected File:**
   - Type: "@"
   - Select: "go.mod"
   - Type: " what is the Go version?"
   - Press Enter
   - ✅ Assistant can reference the file

**Exit:** Press `Ctrl+D`

---

### 2.4 Tool Approval UI

**Test approval dialog for dangerous commands:**

```bash
./bin/spin --model qwen2.5-coder:7b --sandbox workspace-write
```

**Manual Test Steps:**

1. **Trigger Tool Call:**
   - Type: "create a file called test.txt with 'hello' inside"
   - ✅ Approval modal appears
   - ✅ Shows command details
   - ✅ Shows reason for approval

2. **Approve:**
   - Press 'A' for Approve
   - ✅ Command executes
   - ✅ Result shown in chat
   - ✅ File created: `ls test.txt`

3. **Deny:**
   - Type: "delete all files in /tmp"
   - Press 'D' for Deny
   - ✅ Command denied
   - ✅ Assistant informed of denial
   - ✅ No files deleted

4. **Modify:**
   - Type: "list files in /"
   - Press 'M' for Modify
   - Change to: "ls -la"
   - Press Enter
   - ✅ Modified command shown in approval
   - Press 'A' to approve modified
   - ✅ Modified command executes

**Exit:** Press `Ctrl+D`

---

### 2.5 Backtrack Mode (Esc-Esc)

**Test message editing and conversation forking:**

```bash
./bin/spin --model qwen3:1.7b
```

**Manual Test Steps:**

1. **Create Conversation History:**
   - Send: "What is 2+2?"
   - Wait for response
   - Send: "What is 5+5?"
   - Wait for response
   - Send: "What is 10+10?"
   - Wait for response

2. **Enter Backtrack Mode:**
   - Press Esc (input should be empty)
   - Press Esc again
   - ✅ Last user message highlighted
   - ✅ Backtrack mode indicator visible

3. **Navigate History:**
   - Press Esc repeatedly
   - ✅ Highlights step through user messages (10+10 → 5+5 → 2+2)
   - ✅ Visual indicator shows which message

4. **Load Message:**
   - Navigate to "What is 5+5?"
   - Press Enter
   - ✅ Message loaded into input field
   - ✅ Backtrack mode exits
   - ✅ Can edit message

5. **Fork Conversation:**
   - Edit to: "What is 5+6?"
   - Press Enter
   - ✅ Conversation truncated after "What is 2+2?"
   - ✅ New branch starts with "What is 5+6?"
   - ✅ Old "5+5" and "10+10" messages gone

**Exit:** Press `Ctrl+D`

---

### 2.6 Keyboard Shortcuts

**Test all keyboard shortcuts:**

```bash
./bin/spin --model qwen3:1.7b
```

**Manual Test Steps:**

1. **Ctrl+C (Cancel Turn):**
   - Type: "Write a very long story about programming"
   - Press Enter
   - While streaming, press Ctrl+C
   - ✅ Streaming stops
   - ✅ State returns to idle
   - ✅ Can send new message

2. **Ctrl+D (Exit):**
   - Press Ctrl+D
   - ✅ TUI exits gracefully
   - ✅ No errors in terminal

3. **Ctrl+L (Clear Screen):**
   - Restart: `./bin/spin --model qwen3:1.7b`
   - Send a few messages
   - Press Ctrl+L
   - ✅ Screen clears (scrolls to bottom)
   - ✅ History preserved

4. **Ctrl+H or ? (Help):**
   - Press Ctrl+H or ?
   - ✅ Help modal appears
   - ✅ Shows all keyboard shortcuts
   - Press Esc
   - ✅ Help closes

5. **PgUp/PgDn (Scroll):**
   - Send: "List 30 items"
   - Wait for long response
   - Press PgUp
   - ✅ Scrolls up
   - Press PgDn
   - ✅ Scrolls down
   - Press Home
   - ✅ Scrolls to top
   - Press End
   - ✅ Scrolls to bottom

**Exit:** Press `Ctrl+D`

---

### 2.7 Status Bar Updates

**Test status bar real-time updates:**

```bash
./bin/spin --model qwen3:1.7b --sandbox read-only
```

**Manual Test Steps:**

1. **Initial State:**
   - ✅ Model shown: "qwen3:1.7b"
   - ✅ Sandbox: 🔒 read-only
   - ✅ Directory: current path
   - ✅ Tokens: 0 / 0

2. **After First Message:**
   - Send: "Hello"
   - ✅ Token count increases
   - ✅ Format: "123 / 123" (turn / session)

3. **After Second Message:**
   - Send: "Tell me a joke"
   - ✅ Turn tokens reset
   - ✅ Session tokens accumulate
   - ✅ Format: "45 / 168" (example)

4. **Connection Status:**
   - Stop Ollama briefly
   - ✅ Status shows error/disconnected
   - Restart Ollama
   - ✅ Status recovers

**Exit:** Press `Ctrl+D`

---

### 2.8 Error Display & Recovery

**Test error handling in TUI:**

```bash
./bin/spin --model qwen3:1.7b
```

**Manual Test Steps:**

1. **Network Error:**
   - Stop Ollama: `pkill ollama`
   - Send: "test"
   - ✅ Error displayed inline in chat
   - ✅ Error severity shown (critical/warning)
   - ✅ TUI doesn't crash
   - Restart Ollama
   - Send: "test again"
   - ✅ Recovers and works

2. **Invalid Model:**
   - Exit and restart: `./bin/spin --model nonexistent:model`
   - Send: "hello"
   - ✅ Error shown
   - ✅ Clear error message
   - ✅ Suggests valid models (if possible)

3. **Timeout (if implemented):**
   - Configure short timeout
   - Send long task
   - ✅ Timeout error shown
   - ✅ Can retry

**Exit:** Press `Ctrl+D`

---

## Part 3: Integration Tests (Real LLM)

### 3.1 Simple Coding Tasks

**Test AI coding capabilities:**

```bash
./bin/spin --model qwen2.5-coder:7b
```

**Manual Test Steps:**

1. **Write Code:**
   - Type: "Write a Python function to check if a number is prime"
   - ✅ Code generated
   - ✅ Syntax highlighted
   - ✅ Code is correct

2. **File Operations:**
   - Type: "Create a file hello.py with a hello world program"
   - Approve command
   - ✅ File created
   - Verify: `cat hello.py`

3. **Run Code:**
   - Type: "Run the hello.py file"
   - Approve
   - ✅ Code executes
   - ✅ Output shown

**Exit:** Press `Ctrl+D`

---

### 3.2 Multi-Turn Conversation

**Test conversation context:**

```bash
./bin/spin --model qwen3:4b
```

**Manual Test Steps:**

1. **Establish Context:**
   - Send: "My favorite color is blue"
   - Response: (acknowledges)

2. **Reference Context:**
   - Send: "What's my favorite color?"
   - ✅ Response: "Blue" (or similar)
   - ✅ Context maintained

3. **Multi-Step Task:**
   - Send: "Let's write a program. First, tell me what language we should use"
   - Response: (suggests language)
   - Send: "Good choice. Now write a hello world"
   - ✅ Writes code in suggested language
   - ✅ Context from previous message used

**Exit:** Press `Ctrl+D`

---

### 3.3 Tool Use & Execution

**Test real tool calls:**

```bash
./bin/spin --model qwen2.5-coder:7b --sandbox workspace-write
```

**Manual Test Steps:**

1. **File System:**
   - Type: "List all .go files in internal/debug/"
   - Approve `ls` command
   - ✅ Tool call executes
   - ✅ Results shown
   - ✅ AI summarizes results

2. **Create & Modify:**
   - Type: "Create a file test.txt with 'version 1'"
   - Approve
   - Type: "Now change it to 'version 2'"
   - Approve
   - ✅ Both operations work
   - Verify: `cat test.txt` shows "version 2"

3. **Read & Analyze:**
   - Type: "Read the file go.mod and tell me what Go version we're using"
   - Approve read
   - ✅ Reads file
   - ✅ Analyzes and answers

**Exit:** Press `Ctrl+D`

---

## Part 4: Edge Cases & Error Handling

### 4.1 Resource Limits

```bash
# Test 1: Very long prompt
./bin/spin --model qwen3:0.6b
# Type a 10,000 character prompt
# ✅ Handles gracefully (truncates or errors clearly)

# Test 2: Rapid message sending
./bin/spin --model qwen3:0.6b
# Send messages very quickly (Enter, Enter, Enter)
# ✅ Queues or rejects properly
# ✅ No crashes

# Test 3: Empty messages
./bin/spin --model qwen3:0.6b
# Press Enter without typing
# ✅ Ignores or shows validation message
```

---

### 4.2 Signal Handling

```bash
# Test 1: SIGINT during execution
./bin/spin exec --model qwen3:1.7b "long task" &
PID=$!
sleep 2
kill -INT $PID
# ✅ Exits gracefully
# ✅ Exit code 5 (user cancellation)

# Test 2: SIGTERM
./bin/spin exec --model qwen3:1.7b "task" &
PID=$!
kill -TERM $PID
# ✅ Exits gracefully
```

---

### 4.3 Invalid Configurations

```bash
# Test 1: Invalid provider
./bin/spin --provider invalid_provider exec "test"
# ✅ Clear error message

# Test 2: Invalid model
./bin/spin --model "not:a:model" exec "test"
# ✅ Clear error message

# Test 3: Missing config file
./bin/spin --config-file /nonexistent/path config show
# ✅ Error or uses defaults

# Test 4: Malformed config
# Create bad config, try to load
# ✅ Validation catches errors
```

---

## Test Execution Summary Template

```
┌─────────────────────────────────────────────────────────┐
│  SPIN MANUAL TEST EXECUTION REPORT                      │
├─────────────────────────────────────────────────────────┤
│  Date: _______________                                  │
│  Tester: _______________                                │
│  Version: _______________                               │
│  Environment: _______________                           │
├─────────────────────────────────────────────────────────┤
│  Part 1: Non-Interactive Tests                          │
│    1.1 Configuration Tests          [ ] PASS [ ] FAIL   │
│    1.2 MCP Management Tests         [ ] PASS [ ] FAIL   │
│    1.3 Debug Commands Tests         [ ] PASS [ ] FAIL   │
│    1.4 Exec Mode Tests              [ ] PASS [ ] FAIL   │
│    1.5 Version & Help Tests         [ ] PASS [ ] FAIL   │
├─────────────────────────────────────────────────────────┤
│  Part 2: Interactive TUI Tests                          │
│    2.1 TUI Launch & Basic UI        [ ] PASS [ ] FAIL   │
│    2.2 Chat Interface               [ ] PASS [ ] FAIL   │
│    2.3 File Picker (@-trigger)      [ ] PASS [ ] FAIL   │
│    2.4 Tool Approval UI             [ ] PASS [ ] FAIL   │
│    2.5 Backtrack Mode               [ ] PASS [ ] FAIL   │
│    2.6 Keyboard Shortcuts           [ ] PASS [ ] FAIL   │
│    2.7 Status Bar Updates           [ ] PASS [ ] FAIL   │
│    2.8 Error Display & Recovery     [ ] PASS [ ] FAIL   │
├─────────────────────────────────────────────────────────┤
│  Part 3: Integration Tests (Real LLM)                   │
│    3.1 Simple Coding Tasks          [ ] PASS [ ] FAIL   │
│    3.2 Multi-Turn Conversation      [ ] PASS [ ] FAIL   │
│    3.3 Tool Use & Execution         [ ] PASS [ ] FAIL   │
├─────────────────────────────────────────────────────────┤
│  Part 4: Edge Cases                                     │
│    4.1 Resource Limits              [ ] PASS [ ] FAIL   │
│    4.2 Signal Handling              [ ] PASS [ ] FAIL   │
│    4.3 Invalid Configurations       [ ] PASS [ ] FAIL   │
├─────────────────────────────────────────────────────────┤
│  OVERALL RESULT:              [ ] PASS [ ] FAIL         │
│  Critical Issues Found: _______                         │
│  Notes: _________________________________________        │
└─────────────────────────────────────────────────────────┘
```

---

## Automated Test Script

For Part 1 (Non-Interactive), you can run this bash script:

```bash
#!/bin/bash
# test_spin.sh - Automated non-interactive tests

set -e

echo "=== Spin Non-Interactive Test Suite ==="
echo

# 1.1 Config Tests
echo "1.1 Testing configuration..."
./bin/spin config show > /dev/null && echo "✅ config show"
./bin/spin config validate && echo "✅ config validate"
./bin/spin config path && echo "✅ config path"

# 1.2 MCP Tests
echo "1.2 Testing MCP management..."
./bin/spin mcp list && echo "✅ mcp list"
./bin/spin mcp add test-server echo test && echo "✅ mcp add"
./bin/spin mcp remove test-server --yes && echo "✅ mcp remove"

# 1.3 Debug Tests
echo "1.3 Testing debug commands..."
./bin/spin debug events "hello" 2>&1 | grep -q "turn_start" && echo "✅ debug events"
./bin/spin debug sandbox ls 2>&1 | grep -q "only available" && echo "✅ platform check"

# 1.4 Exec Tests
echo "1.4 Testing exec mode..."
timeout 30s ./bin/spin exec --model qwen3:0.6b "say hi" && echo "✅ exec basic"

# 1.5 Version Tests
echo "1.5 Testing version & help..."
./bin/spin --version && echo "✅ version"
./bin/spin --help > /dev/null && echo "✅ help"

echo
echo "=== All automated tests passed! ==="
```

---

## Next Steps for Testing

1. **Run automated script first** (Part 1)
2. **Manually test TUI** (Part 2) - Most critical
3. **Test with real LLM** (Part 3) - Validates integration
4. **Test edge cases** (Part 4) - Ensures robustness

**Recommended Testing Order:**
1. Start with fastest model (`qwen3:0.6b`) for basic tests
2. Use `qwen3:1.7b` or `granite3.3:2b` for TUI tests
3. Use `qwen2.5-coder:7b` for coding tasks
4. Use `qwen3:4b` for multi-turn conversation tests

Ready to begin testing!
