# FRD-20251028000008: LLM Builder Type Safety

**Status**: Complete  
**Created**: 2025-10-28  
**Component**: internal/llm/builder, internal/llm/factory  
**Related**: Phase 6.5 - Empty Interface Elimination

## Problem Statement

The LLM builder and factory use `map[string]interface{}` for provider options, which loses type safety. This occurs in:

1. `builder.go:30` - `Config.Options map[string]interface{}`
2. `factory.go:43` - `ProviderConfig.Options map[string]interface{}`
3. Type assertions scattered across factory for extracting option values

Current known options:
- `auto_tune` (bool) - Whether to enable VRAM auto-tuning
- `vram_headroom_mib` (int) - VRAM headroom in MiB for auto-tuning

## Current Implementation

```go
// builder/builder.go
type Config struct {
    Provider string
    Model    string
    // ...
    Options map[string]interface{}
}

// factory/factory.go
type ProviderConfig struct {
    Type    string
    BaseURL string
    // ...
    Options map[string]interface{}
}

func (f *Factory) shouldAutoTune(options map[string]interface{}) bool {
    if options == nil {
        return true
    }
    if at, ok := options["auto_tune"].(bool); ok {  // Type assertion
        return at
    }
    return true
}
```

## Proposed Solution

### Define ProviderOptions struct

Create a typed struct for known provider options:

```go
// ProviderOptions contains provider-specific configuration options.
type ProviderOptions struct {
    // AutoTune enables automatic VRAM-based optimization (Ollama only).
    // Default: true
    AutoTune bool
    
    // VRAMHeadroomMiB specifies VRAM headroom in MiB for auto-tuning (Ollama only).
    // Default: 1024 (1GB)
    VRAMHeadroomMiB int
}

// DefaultProviderOptions returns the default options.
func DefaultProviderOptions() ProviderOptions {
    return ProviderOptions{
        AutoTune:        true,
        VRAMHeadroomMiB: 1024,
    }
}
```

### Update Config structs

```go
// builder/builder.go
type Config struct {
    Provider string
    Model    string
    BaseURL  string
    Timeout  time.Duration
    KeyName  string
    APIKey   string
    Options  ProviderOptions  // Typed instead of map
}

// factory/factory.go
type ProviderConfig struct {
    Type    string
    BaseURL string
    Model   string
    Timeout time.Duration
    KeyName string
    APIKey  string
    Options ProviderOptions  // Typed instead of map
}
```

### Update factory methods

```go
func (f *Factory) shouldAutoTune(options ProviderOptions) bool {
    return options.AutoTune
}

func (f *Factory) extractVRAMHeadroom(options ProviderOptions) int64 {
    return int64(options.VRAMHeadroomMiB) * 1024 * 1024
}
```

### Update config merging

```go
// builder/builder.go
func (b *Builder) mergeConfig(explicit Config) Config {
    merged := explicit
    
    // ... existing code ...
    
    // Provider options from config file
    if b.configLoader.IsSet("llm.auto_tune") {
        merged.Options.AutoTune = b.configLoader.GetBool("llm.auto_tune")
    }
    if b.configLoader.IsSet("llm.vram.headroom_mib") {
        merged.Options.VRAMHeadroomMiB = b.configLoader.GetInt("llm.vram.headroom_mib")
    }
    
    return merged
}
```

## Benefits

1. **Type Safety**: Compiler enforces correct option types
2. **IDE Support**: Auto-completion for option fields
3. **Documentation**: Struct clearly documents available options
4. **Validation**: Can add validation methods to ProviderOptions
5. **Default Values**: Clear default values in one place
6. **No Type Assertions**: Eliminates error-prone type assertions

## Interface{} Eliminated

- 2 occurrences in production code (Config.Options, ProviderConfig.Options)
- ~8 occurrences in test code

## Testing Strategy

- Update all existing tests to use ProviderOptions
- Test default values
- Test config file loading
- Test explicit overrides
- Test auto-tune logic with typed options
- Ensure all 3 test packages pass (builder, factory)

## Migration Impact

- **Breaking Changes**: None - internal types only
- **API Compatibility**: Builder.Build() signature unchanged (internal Config type only)
- **Performance**: No impact (struct access faster than map lookups)

## Implementation Checklist

- [x] Define ProviderOptions struct in factory package
- [x] Update builder.Config to use ProviderOptions
- [x] Update factory.ProviderConfig to use ProviderOptions
- [x] Update mergeConfig to populate typed options
- [x] Update shouldAutoTune to use typed parameter
- [x] Update extractVRAMHeadroom to use typed parameter
- [x] Update createOllamaProvider to pass typed options
- [x] Update all tests (builder + factory)
- [x] Run all tests - all pass
- [x] Run go vet - clean
- [x] Update documentation
- [x] Update roadmap

## Implementation Results

**Files Modified**:
- `internal/llm/factory/factory.go` - Added ProviderOptions struct, updated ProviderConfig (+21 lines, -30 lines)
- `internal/llm/builder/builder.go` - Updated Config to use ProviderOptions, simplified mergeConfig (+3 lines, -18 lines)
- `internal/llm/factory/factory_test.go` - Updated test to use typed options (+8 lines, -8 lines)

**Code Changes**:
1. Added `ProviderOptions` struct with `AutoTune` and `VRAMHeadroomMiB` fields
2. Simplified `shouldAutoTune()` from 9 lines to 1 line (eliminated type assertions)
3. Updated `extractVRAMHeadroom()` to handle zero value with default (3 lines, was 17 lines with type assertions)
4. Simplified `mergeConfig()` options handling from 22 lines to 12 lines
5. Eliminated `DefaultProviderOptions()` helper (deadcode) - defaults now handled inline

**Note**: Initially added `DefaultProviderOptions()` but removed it to avoid deadcode. Default values now handled:
- `AutoTune`: defaults to `true` in builder's mergeConfig if not explicitly set
- `VRAMHeadroomMiB`: defaults to `1024` in factory's extractVRAMHeadroom if zero

**Interface{} Eliminated**: 2 occurrences in production code
- `builder.Config.Options`
- `factory.ProviderConfig.Options`

**Test Results**: 
- Builder: 9 tests pass, 95.2% coverage
- Factory: 28 tests pass, 91.2% coverage
- Total code reduction: ~45 lines (eliminated type assertions and complex logic)

**Benefits Achieved**:
- Zero type assertions in factory code
- Clear default values (AutoTune=true, VRAMHeadroomMiB=1024)
- IDE auto-completion for options
- Compiler enforces correct types
- Easier to extend with new options

## Future Extensions

If new provider-specific options are needed in the future:
- Add new fields to ProviderOptions with appropriate defaults
- Document which providers support which options
- Consider provider-specific option structs if needed
