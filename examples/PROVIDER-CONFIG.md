# Spin Provider Configuration Guide

This guide shows how to configure Spin to work with different LLM providers.

## Quick Start

### 1. Local Ollama (No auth required)

```bash
# Pull a model first
ollama pull llama3.1

# Use via flags
spin exec --provider ollama --model llama3.1 "fix linting errors"

# Or copy config-ollama.yaml to ~/.spin/spin.yaml
cp examples/config-ollama.yaml ~/.spin/spin.yaml
spin exec "fix linting errors"
```

### 2. OpenAI (Auth required)

```bash
# Option 1: Environment variable (easiest)
export OPENAI_API_KEY=sk-...
spin exec --provider openai --model gpt-4o "refactor this code"

# Option 2: Keystore (recommended, secure)
spin config set-key my-openai-key sk-...
spin exec --provider openai --model gpt-4o --key-name my-openai-key "analyze code"

# Option 3: Config file
cp examples/config-openai.yaml ~/.spin/spin.yaml
# Edit the file to add your key_name or api_key
spin exec "run all tests"
```

### 3. LMStudio (Local, no auth)

```bash
# Start LMStudio server first (http://localhost:1234)

# Use via flags
spin exec --provider lmstudio --model codellama-13b "explain this function"

# Or use config file
cp examples/config-lmstudio.yaml ~/.spin/spin.yaml
spin exec "add error handling"
```

### 4. Custom OpenAI-Compatible API

```bash
# Works with Together AI, Anyscale, etc.
export OPENAI_API_KEY=your-api-key

spin exec \
  --provider openai-compatible \
  --model mixtral-8x7b-instruct \
  --base-url https://api.together.xyz/v1 \
  "optimize performance"
```

## Configuration Precedence

Configuration is merged in this order (highest to lowest priority):

1. **Command-line flags** (highest)
2. **Environment variables**
3. **Config file** (`~/.spin/spin.yaml`)
4. **Built-in defaults** (lowest)

## Authentication Methods

### Keystore (Recommended) - Most Secure

```bash
# Store credential securely in OS keyring
spin config set-key my-openai-key sk-...

# Use via flag
spin exec --provider openai --key-name my-openai-key "task"

# Or via config file
llm:
  key_name: my-openai-key
```

### Environment Variable - Easy for CI/CD

```bash
export OPENAI_API_KEY=sk-...
spin exec --provider openai --model gpt-4o "task"
```

### Direct API Key - Deprecated ⚠️

```bash
spin exec --provider openai --api-key sk-... "task"
# Shows deprecation warning
```

## Provider-Specific Details

### Ollama
- **URL:** `http://localhost:11434`
- **Auth:** None
- **Setup:** `ollama serve && ollama pull llama3.1`
- **Models:** llama3.1, mixtral, codellama, etc.

### LMStudio
- **URL:** `http://localhost:1234/v1`
- **Auth:** None
- **Setup:** Start server in LMStudio app, select model
- **Models:** Any supported by LMStudio

### OpenAI
- **URL:** `https://api.openai.com/v1`
- **Auth:** Required (API key)
- **Get key:** https://platform.openai.com/api-keys
- **Models:** gpt-4o, gpt-4-turbo, gpt-3.5-turbo

### Anthropic
- **URL:** `https://api.anthropic.com/v1`
- **Auth:** Required (API key)
- **Get key:** https://console.anthropic.com/
- **Models:** claude-3-opus, claude-3-sonnet, claude-3-haiku
- **Env var:** `ANTHROPIC_API_KEY`

### OpenAI-Compatible
- **URL:** Custom (provider-specific)
- **Auth:** Usually required
- **Examples:**
  - Together AI: `https://api.together.xyz/v1`
  - Anyscale: `https://api.endpoints.anyscale.com/v1`
  - Perplexity: `https://api.perplexity.ai`
  - Local (vLLM, text-generation-webui): Custom

## Example Configurations

See the following example files in this directory:

- `config-ollama.yaml` - Local Ollama setup
- `config-openai.yaml` - OpenAI with keystore
- `config-lmstudio.yaml` - Local LMStudio setup
- `config-custom.yaml` - Custom OpenAI-compatible API

## Troubleshooting

### "authentication required" error

```bash
# Set environment variable
export OPENAI_API_KEY=sk-...

# Or use keystore
spin config set-key my-key sk-...
spin exec --key-name my-key ...
```

### "model is required" error

```bash
spin exec --model llama3.1 "task"
# or
export SPIN_MODEL=llama3.1
```

### Connection errors

1. Check provider is running (local providers)
2. Verify base URL: `curl http://localhost:11434/api/version`
3. Increase timeout: `spin exec --timeout 5m "task"`

## CI/CD Usage

```yaml
# GitHub Actions example
- name: Fix linting
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: |
    spin exec \
      --provider openai \
      --model gpt-4o \
      --auto-approve \
      "Fix all linting errors"
```

## Security Best Practices

1. ✅ Use keystore for credentials
2. ✅ Use environment variables in CI/CD
3. ✅ Never commit API keys to git
4. ✅ Add `spin.yaml` to `.gitignore` if it contains keys
5. ✅ Use `read-only` sandbox mode when possible

For more examples, see the example config files in this directory.
