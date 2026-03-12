package conversation

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/memory"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// MemoryService holds the memory stores for context offloading.
type MemoryService struct {
	scratchpad *memory.Scratchpad
	persistent *memory.PersistentStore
}

// NewMemoryService creates a new memory service with the configured stores.
func NewMemoryService(scratchpad *memory.Scratchpad, persistent *memory.PersistentStore) *MemoryService {
	return &MemoryService{
		scratchpad: scratchpad,
		persistent: persistent,
	}
}

// Scratchpad returns the session-scoped scratchpad store.
func (m *MemoryService) Scratchpad() *memory.Scratchpad {
	return m.scratchpad
}

// Persistent returns the cross-session persistent store.
func (m *MemoryService) Persistent() *memory.PersistentStore {
	return m.persistent
}

// NewAutoOffloader creates an auto-offloader for this memory service.
// Returns nil if no stores are configured.
func (m *MemoryService) NewAutoOffloader(threshold float64) *memory.AutoOffloader {
	if m.scratchpad == nil && m.persistent == nil {
		return nil
	}

	return memory.NewAutoOffloader(memory.AutoOffloaderConfig{
		Scratchpad: m.scratchpad,
		Persistent: m.persistent,
		Threshold:  threshold,
	})
}

// NewSessionHandoff creates a session handoff manager for this memory service.
// Returns nil if persistent store is not configured.
// If summarizer is nil, a SimpleSummarizer with default settings is used.
func (m *MemoryService) NewSessionHandoff(summarizer memory.Summarizer) *memory.SessionHandoff {
	if m.persistent == nil {
		return nil
	}

	if summarizer == nil {
		summarizer = memory.NewSimpleSummarizer(500)
	}

	return memory.NewSessionHandoff(m.persistent, summarizer)
}

// initializeMemory creates memory stores based on configuration.
// This is called during Build() to set up memory services.
func (b *Builder) initializeMemory(sessionID string) error {
	if b.cfg == nil {
		return nil
	}

	memCfg := b.cfg.Memory

	var (
		scratchpad *memory.Scratchpad
		persistent *memory.PersistentStore
		err        error
	)

	// Initialize scratchpad if enabled.

	if memCfg.Scratchpad.Enabled {
		maxEntries := memCfg.Scratchpad.MaxEntries
		if maxEntries <= 0 {
			maxEntries = 50 // Default.
		}

		scratchpad = memory.NewScratchpad(sessionID, maxEntries)
		if b.logger != nil {
			b.logger.Debug("scratchpad initialized", "session_id", sessionID, "max_entries", maxEntries)
		}
	}

	// Initialize persistent store if enabled.
	if memCfg.Persistent.Enabled {
		basePath := memCfg.Persistent.BasePath
		if basePath == "" {
			basePath = "~/.spin/memory"
		}

		persistent, err = memory.NewPersistentStore(basePath)
		if err != nil {
			return fmt.Errorf("initialize persistent memory: %w", err)
		}

		if b.logger != nil {
			b.logger.Debug("persistent memory initialized", "base_path", basePath)
		}
	}

	// Only create service if at least one store is enabled.
	if scratchpad != nil || persistent != nil {
		b.memoryService = NewMemoryService(scratchpad, persistent)
	}

	return nil
}

// registerMemoryTools adds memory tools to the registry.
func (b *Builder) registerMemoryTools(registry *tools.Registry) error {
	if b.memoryService == nil {
		return nil
	}

	if err := b.registerScratchpadTool(registry); err != nil {
		return err
	}

	return b.registerPersistentMemoryTool(registry)
}

// registerScratchpadTool registers the scratchpad tool if available.
func (b *Builder) registerScratchpadTool(registry *tools.Registry) error {
	if b.memoryService.scratchpad == nil {
		return nil
	}

	scratchpadTool := tools.NewScratchpadTool(b.memoryService.scratchpad)
	if scratchpadTool == nil {
		return nil
	}

	if err := registry.RegisterOrReplace(scratchpadTool); err != nil {
		return fmt.Errorf("register scratchpad tool: %w", err)
	}

	if b.logger != nil {
		b.logger.Debug("scratchpad tool registered")
	}

	return nil
}

// registerPersistentMemoryTool registers the persistent memory tool if available.
func (b *Builder) registerPersistentMemoryTool(registry *tools.Registry) error {
	if b.memoryService.persistent == nil {
		return nil
	}

	memoryTool := tools.NewMemoryTool(b.memoryService.persistent)
	if memoryTool == nil {
		return nil
	}

	if err := registry.RegisterOrReplace(memoryTool); err != nil {
		return fmt.Errorf("register memory tool: %w", err)
	}

	if b.logger != nil {
		b.logger.Debug("memory tool registered")
	}

	return nil
}
