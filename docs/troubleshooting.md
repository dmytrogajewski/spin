## NAME

Troubleshooting - solving common issues with Spin

## DESCRIPTION

This guide helps you diagnose and fix common issues when using Spin. Each section covers a specific problem with symptoms, causes, and solutions.

## CONFIGURATION ISSUES

### "invalid configuration: model is required"

**Symptoms:**
- Spin fails to start with error: `invalid configuration: model is required`
- Agent cannot initialize

**Causes:**
- Model not specified in config file, environment variable, or command-line flag
- Configuration file not being read correctly
- Environment variable override not working

**Solutions:**

1. **Check configuration file:**
   ```bash
   spin config show
   # Verify llm.model is set
   ```

2. **Specify model via flag:**
   ```bash
   spin --model qwen3:0.6b --provider ollama
   ```

3. **Set environment variable:**
   ```bash
   export SPIN_MODEL=qwen3:0.6b
   spin --provider ollama
   ```

4. **Validate configuration:**
   ```bash
   spin config validate
   ```

5. **Check config file location:**
   ```bash
   spin config path
   # Ensure file exists and is readable
   ```

### "authentication required" or "401 Unauthorized"

**Symptoms:**
- API calls fail with authentication errors
- Provider returns 401 status code

**Causes:**
- API key not set or incorrect
- Key expired or revoked
- Wrong authentication method for provider

**Solutions:**

1. **Check keystore:**
   ```bash
   # Verify key exists
   spin config get llm.key_name
   ```

2. **Set key via environment variable:**
   ```bash
   export OPENAI_API_KEY=sk-...
   spin --provider openai --model gpt-4o
   ```

3. **Store key in keystore:**
   ```bash
   spin config set-key my-key sk-...
   spin --provider openai --model gpt-4o --key-name my-key
   ```

4. **Verify key is valid:**
   ```bash
   # Test with curl for OpenAI
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"
   ```

### Configuration file not found or ignored

**Symptoms:**
- Changes to config file don't take effect
- Spin uses defaults instead of config

**Solutions:**

1. **Check config file location:**
   ```bash
   spin config path
   # Default: ~/.spin/spin.yaml or ~/.spin/config.yaml
   ```

2. **Specify config file explicitly:**
   ```bash
   spin --config-file /path/to/config.yaml
   ```

3. **Validate configuration syntax:**
   ```bash
   spin config validate --file ~/.spin/spin.yaml
   ```

4. **Check file permissions:**
   ```bash
   ls -la ~/.spin/spin.yaml
   # Ensure file is readable
   ```

## PROVIDER CONNECTION ISSUES

### "connection refused" or "connection timeout"

**Symptoms:**
- Cannot connect to LLM provider
- Timeout errors when calling provider

**Causes:**
- Provider server not running (for local providers)
- Wrong base URL
- Network connectivity issues
- Firewall blocking connections

**Solutions:**

1. **For local providers (Ollama, LM Studio):**
   ```bash
   # Check if Ollama is running
   curl http://localhost:11434/api/version
   
   # Start Ollama if not running
   ollama serve
   
   # Check if LM Studio server is running
   curl http://localhost:1234/v1/models
   ```

2. **Verify base URL:**
   ```bash
   # Check config
   spin config get llm.base_url
   
   # Override with flag
   spin --base-url http://localhost:11434 --provider ollama
   ```

3. **Test connectivity:**
   ```bash
   # Test Ollama
   curl http://localhost:11434/api/tags
   
   # Test OpenAI
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"
   ```

4. **Increase timeout:**
   ```bash
   spin --timeout 10m "your prompt"
   ```

### "model not found" or "model does not exist"

**Symptoms:**
- Provider returns error about model not existing
- Model name is incorrect

**Solutions:**

1. **List available models:**
   ```bash
   # For Ollama
   ollama list
   
   # Pull model if missing
   ollama pull qwen3:0.6b
   ```

2. **Check model name spelling:**
   ```bash
   # Verify exact model name
   spin config get llm.model
   ```

3. **Use correct model name for provider:**
   - Ollama: `qwen3:0.6b`, `llama3.1`, `mixtral`
   - OpenAI: `gpt-4o`, `gpt-4-turbo`, `gpt-3.5-turbo`
   - Anthropic: `claude-3-5-sonnet-20241022`, `claude-3-opus`

## AGENT EXECUTION ISSUES

### Agent gets stuck in "thinking" state

**Symptoms:**
- Agent shows "Thinking..." but never progresses
- No tool calls or responses
- Process appears hung

**Causes:**
- LLM timeout too short
- Provider slow to respond
- Infinite loop in agent logic
- Context cancellation issues

**Solutions:**

1. **Increase timeout:**
   ```bash
   spin --timeout 10m "your prompt"
   ```

2. **Check LLM provider status:**
   ```bash
   # For Ollama, check if model is loaded
   ollama ps
   ```

3. **Try a simpler prompt:**
   ```bash
   spin "say hello"
   ```

4. **Use a faster model:**
   ```bash
   # Switch to smaller/faster model
   spin --model qwen3:0.6b --provider ollama
   ```

5. **Check for cycle detection:**
   - Agent may be detecting repeated tool calls
   - Try a different approach or restart

### Tool execution fails

**Symptoms:**
- Tools are called but fail to execute
- "Failed to execute tool" errors
- Tool results not appearing

**Solutions:**

1. **Check sandbox mode:**
   ```bash
   # Verify sandbox allows the operation
   spin config get security.sandbox.mode
   
   # Try with different sandbox mode
   spin --sandbox workspace-write
   ```

2. **Check tool permissions:**
   ```bash
   # Verify file permissions
   ls -la /path/to/file
   ```

3. **Enable debug mode:**
   ```bash
   spin --debug "your prompt"
   # Check logs for detailed error messages
   ```

4. **Verify working directory:**
   ```bash
   spin --cd /path/to/workspace "your prompt"
   ```

### Approval requests not appearing

**Symptoms:**
- Dangerous operations execute without approval
- No approval dialogs shown

**Solutions:**

1. **Check approval service configuration:**
   ```bash
   spin config get security.policy_file
   ```

2. **Verify you're not using `--auto-approve`:**
   ```bash
   # Remove --auto-approve flag
   spin "your prompt"  # Will prompt for approvals
   ```

3. **Check approval policies:**
   ```bash
   # List existing policies
   spin approval list
   
   # Clear policies if needed
   spin approval clear
   ```

## PERFORMANCE ISSUES

### Slow response times

**Symptoms:**
- Agent takes a long time to respond
- Tool execution is slow

**Solutions:**

1. **Use a faster model:**
   ```bash
   # Smaller models are faster
   spin --model qwen3:0.6b --provider ollama
   ```

2. **Reduce context size:**
   ```bash
   # Use compact mode
   spin --mode compact "your prompt"
   ```

3. **Check provider performance:**
   ```bash
   # For Ollama, check model size
   ollama list
   # Smaller models = faster responses
   ```

4. **Increase timeout if needed:**
   ```bash
   spin --timeout 10m "complex task"
   ```

### High memory usage

**Symptoms:**
- Spin uses excessive memory
- System becomes slow

**Solutions:**

1. **Use smaller models:**
   ```bash
   # Smaller models use less memory
   spin --model qwen3:0.6b
   ```

2. **Reduce max tokens:**
   ```yaml
   llm:
     max_tokens: 2048  # Reduce from default
   ```

3. **Limit conversation history:**
   ```yaml
   task:
     max_turns: 20  # Reduce from default 50
   ```

## IDE/ACP INTEGRATION ISSUES

### IDE cannot connect to Spin

**Symptoms:**
- IDE shows connection error
- Spin ACP server not responding

**Solutions:**

1. **Verify Spin binary is in PATH:**
   ```bash
   which spin
   # Or use full path in IDE config
   ```

2. **Test ACP server manually:**
   ```bash
   spin acp --provider ollama --model qwen3:0.6b
   # Should start without errors
   ```

3. **Check IDE configuration:**
   - Verify agent command: `spin acp --provider ollama --model qwen3:0.6b`
   - Check workspace path is correct
   - Ensure provider is accessible

4. **Check for port conflicts:**
   - ACP uses stdin/stdout, no ports needed
   - Verify no other processes interfering

### Session not persisting

**Symptoms:**
- Conversation history lost after IDE restart
- Session cannot be loaded

**Solutions:**

1. **Check session storage:**
   ```bash
   # Verify session directory exists
   ls -la ~/.spin/sessions/
   ```

2. **Check session storage configuration:**
   ```yaml
   storage:
     session_dir: ~/.spin/sessions
   ```

3. **Verify permissions:**
   ```bash
   # Ensure directory is writable
   chmod 755 ~/.spin/sessions
   ```

## GENERAL TROUBLESHOOTING STEPS

### Enable Debug Mode

Get detailed logging for diagnosis:

```bash
# TUI mode with debug
spin --debug

# Exec mode with debug
spin exec --debug "your prompt"

# ACP mode with debug (check IDE logs)
spin acp --debug --provider ollama --model qwen3:0.6b
```

### Check Logs

Spin logs to stderr. Capture logs for analysis:

```bash
# Redirect logs to file
spin --debug "your prompt" 2> spin.log

# View recent errors
tail -f spin.log
```

### Validate Environment

Run a simple test to verify setup:

```bash
# Test basic functionality
spin --provider ollama --model qwen3:0.6b "say hello"

# Test config loading
spin config show

# Test provider connection
spin exec --provider ollama --model qwen3:0.6b "what is 2+2?"
```

### Reset Configuration

Start fresh if configuration is corrupted:

```bash
# Backup current config
cp ~/.spin/spin.yaml ~/.spin/spin.yaml.backup

# Remove config (will use defaults)
rm ~/.spin/spin.yaml

# Or edit to minimal config
cat > ~/.spin/spin.yaml <<EOF
version: "2.0"
llm:
  provider: ollama
  model: qwen3:0.6b
EOF
```

## GETTING HELP

If you're still experiencing issues:

1. **Check existing issues:**
   - GitHub Issues: https://github.com/dmytrogajewski/spin/issues

2. **Gather diagnostic information:**
   ```bash
   # System info
   uname -a
   go version
   
   # Spin version
   spin --version
   
   # Configuration (redacted)
   spin config show
   
   # Test with debug
   spin --debug "test" 2>&1 | tee debug.log
   ```

3. **Create a minimal reproduction:**
   - Smallest config that reproduces the issue
   - Exact command that fails
   - Expected vs actual behavior

## RELATED DOCUMENTS

- `docs/configuration.md` – configuration guide
- `docs/job-local-agent.md` – local agent usage
- `docs/job-ci-automation.md` – CI/CD usage
- `docs/job-acp-ide.md` – IDE integration
- `README.md` – main documentation

