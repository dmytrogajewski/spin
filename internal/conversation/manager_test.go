package conversation

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

// mockConversation creates a minimal conversation for testing.
func mockConversation(id string) *Conversation {
	hist := history.NewHistory(8192, &tokenizer.SimpleTokenizer{})

	return &Conversation{
		id:       id,
		history:  hist,
		taskMode: "regular",
	}
}

// mockFactory creates a simple factory for testing.
func mockFactory() ConversationFactory {
	return func(ctx context.Context, sessionID string, workDir string) (*Conversation, error) {
		return mockConversation(sessionID), nil
	}
}

// errorFactory creates a factory that always returns an error.
func errorFactory(errMsg string) ConversationFactory {
	return func(ctx context.Context, sessionID string, workDir string) (*Conversation, error) {
		return nil, fmt.Errorf("%s", errMsg)
	}
}

func TestManager_NewManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mgr, err := NewManager(ManagerConfig{
			Factory: mockFactory(),
		})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		if mgr == nil {
			t.Fatal("manager should not be nil")
		}
	})

	t.Run("nil factory", func(t *testing.T) {
		_, err := NewManager(ManagerConfig{})
		if err == nil {
			t.Fatal("NewManager should fail with nil factory")
		}
	})
}

func TestManager_GetOrCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("create new", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

		conv, err := mgr.GetOrCreate(ctx, "session-1", "/tmp")
		if err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

		if conv == nil {
			t.Fatal("conversation should not be nil")
		}

		if conv.ID() != "session-1" {
			t.Errorf("ID mismatch: got %q, want %q", conv.ID(), "session-1")
		}
	})

	t.Run("get existing", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

		conv1, _ := mgr.GetOrCreate(ctx, "session-1", "/tmp")
		conv2, _ := mgr.GetOrCreate(ctx, "session-1", "/tmp")

		if conv1 != conv2 {
			t.Error("should return same conversation instance")
		}
	})

	t.Run("empty session ID", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

		_, err := mgr.GetOrCreate(ctx, "", "/tmp")
		if err == nil {
			t.Fatal("should fail with empty session ID")
		}
	})

	t.Run("factory error", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: errorFactory("factory failed")})

		_, err := mgr.GetOrCreate(ctx, "session-1", "/tmp")
		if err == nil {
			t.Fatal("should fail when factory fails")
		}
	})
}

func TestManager_Get(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	t.Run("not found", func(t *testing.T) {
		conv, ok := mgr.Get("non-existent")
		if ok {
			t.Error("should not find non-existent conversation")
		}

		if conv != nil {
			t.Error("conversation should be nil")
		}
	})

	t.Run("found", func(t *testing.T) {
		mgr.GetOrCreate(ctx, "session-1", "/tmp")

		conv, ok := mgr.Get("session-1")
		if !ok {
			t.Error("should find existing conversation")
		}

		if conv == nil {
			t.Error("conversation should not be nil")
		}
	})
}

func TestManager_Remove(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	t.Run("remove existing", func(t *testing.T) {
		mgr.GetOrCreate(ctx, "session-1", "/tmp")

		err := mgr.Remove("session-1")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		_, ok := mgr.Get("session-1")
		if ok {
			t.Error("conversation should be removed")
		}
	})

	t.Run("remove non-existent", func(t *testing.T) {
		err := mgr.Remove("non-existent")
		if err == nil {
			t.Fatal("should fail when removing non-existent")
		}
	})
}

func TestManager_Cancel(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	// Create conversation and set a cancel function.
	conv, _ := mgr.GetOrCreate(ctx, "session-1", "/tmp")

	canceled := false

	conv.SetCancel(func() { canceled = true })

	// Cancel via manager.
	mgr.Cancel("session-1")

	if !canceled {
		t.Error("cancel function should have been called")
	}

	// Cancel non-existent should not panic.
	mgr.Cancel("non-existent")
}

func TestManager_List(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	t.Run("empty", func(t *testing.T) {
		list := mgr.List()
		if len(list) != 0 {
			t.Errorf("should be empty, got %d", len(list))
		}
	})

	t.Run("with conversations", func(t *testing.T) {
		mgr.GetOrCreate(ctx, "session-1", "/tmp")
		mgr.GetOrCreate(ctx, "session-2", "/tmp")

		list := mgr.List()
		if len(list) != 2 {
			t.Errorf("should have 2 conversations, got %d", len(list))
		}
	})
}

func TestManager_Count(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	if mgr.Count() != 0 {
		t.Error("should start with 0 conversations")
	}

	mgr.GetOrCreate(ctx, "session-1", "/tmp")

	if mgr.Count() != 1 {
		t.Error("should have 1 conversation")
	}

	mgr.GetOrCreate(ctx, "session-2", "/tmp")

	if mgr.Count() != 2 {
		t.Error("should have 2 conversations")
	}

	mgr.Remove("session-1")

	if mgr.Count() != 1 {
		t.Error("should have 1 conversation after remove")
	}
}

func TestManager_Close(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	mgr.GetOrCreate(ctx, "session-1", "/tmp")
	mgr.GetOrCreate(ctx, "session-2", "/tmp")

	err := mgr.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if mgr.Count() != 0 {
		t.Error("should have no conversations after close")
	}
}

func TestManager_SetTaskMode(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	t.Run("set mode", func(t *testing.T) {
		mgr.GetOrCreate(ctx, "session-1", "/tmp")

		err := mgr.SetTaskMode("session-1", "compact")
		if err != nil {
			t.Fatalf("SetTaskMode failed: %v", err)
		}

		mode, _ := mgr.GetTaskMode("session-1")
		if mode != "compact" {
			t.Errorf("mode mismatch: got %q, want %q", mode, "compact")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := mgr.SetTaskMode("non-existent", "compact")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})
}

func TestManager_GetTaskMode(t *testing.T) {
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	t.Run("not found", func(t *testing.T) {
		_, err := mgr.GetTaskMode("non-existent")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})
}

func TestManager_SaveAndLoad(t *testing.T) {
	// Create temp directory for storage.
	tmpDir, err := os.MkdirTemp("", "manager-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	histStorage, err := history.NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create history storage: %v", err)
	}

	ctx := context.Background()

	t.Run("save and load", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{
			Factory:        mockFactory(),
			HistoryStorage: histStorage,
		})

		// Create conversation and add messages.
		conv, _ := mgr.GetOrCreate(ctx, "session-save", "/tmp")
		conv.history.AddUserMessage("Hello!")
		conv.history.AddMessage(message.Message{
			Role:    message.RoleAssistant,
			Content: "Hi there!",
		})

		// Save.
		err := mgr.Save("session-save")
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Create new manager and load.
		mgr2, _ := NewManager(ManagerConfig{
			Factory:        mockFactory(),
			HistoryStorage: histStorage,
		})

		conv2, err := mgr2.Load(ctx, "session-save", "/tmp")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Verify messages loaded.
		msgs := conv2.GetHistoryMessages()
		// Should have system message + 2 added messages.
		if len(msgs) < 2 {
			t.Errorf("should have at least 2 messages, got %d", len(msgs))
		}
	})

	t.Run("load non-existent", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{
			Factory:        mockFactory(),
			HistoryStorage: histStorage,
		})

		_, err := mgr.Load(ctx, "non-existent", "/tmp")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})

	t.Run("save without storage", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})
		mgr.GetOrCreate(ctx, "session-1", "/tmp")

		err := mgr.Save("session-1")
		if err == nil {
			t.Fatal("should fail without storage configured")
		}
	})

	t.Run("load without storage", func(t *testing.T) {
		mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

		_, err := mgr.Load(ctx, "session-1", "/tmp")
		if err == nil {
			t.Fatal("should fail without storage configured")
		}
	})
}

func TestManager_Concurrent(t *testing.T) {
	mgr, _ := NewManager(ManagerConfig{Factory: mockFactory()})

	var wg sync.WaitGroup

	numGoroutines := 100

	// Concurrent GetOrCreate.
	for i := range numGoroutines {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("session-%d", n%10) // 10 unique sessions.

			_, err := mgr.GetOrCreate(context.Background(), sessionID, "/tmp")
			if err != nil {
				t.Errorf("GetOrCreate failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Should have exactly 10 sessions.
	if mgr.Count() != 10 {
		t.Errorf("should have 10 sessions, got %d", mgr.Count())
	}

	// Concurrent Get.
	for i := range numGoroutines {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("session-%d", n%10)
			mgr.Get(sessionID)
		}(i)
	}

	wg.Wait()

	// Concurrent List.
	for range numGoroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			mgr.List()
		}()
	}

	wg.Wait()
}
