package conversation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

var errError = errors.New("error")

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
func mockFactory() Factory {
	return func(_ context.Context, sessionID string, _ string) (*Conversation, error) {
		return mockConversation(sessionID), nil
	}
}

// errorFactory creates a factory that always returns an error.
func errorFactory(errMsg string) Factory {
	return func(_ context.Context, _ string, _ string) (*Conversation, error) {
return nil, fmt.Errorf("%s: %w", errMsg, errError)
	}
}

func TestManager_NewManager(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		_, err := NewManager(ManagerConfig{})
		if err == nil {
			t.Fatal("NewManager should fail with nil factory")
		}
	})
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()

	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	return mgr
}

func TestManager_GetOrCreate_CreateNew(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)
	conv, err := mgr.GetOrCreate(context.Background(), "session-1", "/tmp")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if conv == nil {
		t.Fatal("conversation should not be nil")
	}
	if conv.ID() != "session-1" {
		t.Errorf("ID mismatch: got %q, want %q", conv.ID(), "session-1")
	}
}

func TestManager_GetOrCreate_GetExisting(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)
	ctx := context.Background()
	conv1, err := mgr.GetOrCreate(ctx, "session-1", "/tmp")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	conv2, err := mgr.GetOrCreate(ctx, "session-1", "/tmp")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if conv1 != conv2 {
		t.Error("should return same conversation instance")
	}
}

func TestManager_GetOrCreate_EmptySessionID(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)
	_, err := mgr.GetOrCreate(context.Background(), "", "/tmp")
	if err == nil {
		t.Fatal("should fail with empty session ID")
	}
}

func TestManager_GetOrCreate_FactoryError(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(ManagerConfig{Factory: errorFactory("factory failed")})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	_, err = mgr.GetOrCreate(context.Background(), "session-1", "/tmp")
	if err == nil {
		t.Fatal("should fail when factory fails")
	}
}

func TestManager_Get(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		conv, ok := mgr.Get("non-existent")
		if ok {
			t.Error("should not find non-existent conversation")
		}

		if conv != nil {
			t.Error("conversation should be nil")
		}
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		_, err = mgr.GetOrCreate(ctx, "session-1", "/tmp")
		if err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

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
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	t.Run("remove existing", func(t *testing.T) {
		t.Parallel()

		_, err = mgr.GetOrCreate(ctx, "session-1", "/tmp")
		if err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

		err = mgr.Remove("session-1")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		_, ok := mgr.Get("session-1")
		if ok {
			t.Error("conversation should be removed")
		}
	})

	t.Run("remove non-existent", func(t *testing.T) {
		t.Parallel()

		err = mgr.Remove("non-existent")
		if err == nil {
			t.Fatal("should fail when removing non-existent")
		}
	})
}

func TestManager_Cancel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create conversation and set a cancel function.
	conv, err := mgr.GetOrCreate(ctx, "session-1", "/tmp")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

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
	t.Parallel()

	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		list := mgr.List()
		if len(list) != 0 {
			t.Errorf("should be empty, got %d", len(list))
		}
	})

	t.Run("with conversations", func(t *testing.T) {
		t.Parallel()

		mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		if _, err = mgr.GetOrCreate(ctx, "session-1", "/tmp"); err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}
		if _, err = mgr.GetOrCreate(ctx, "session-2", "/tmp"); err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

		list := mgr.List()
		if len(list) != 2 {
			t.Errorf("should have 2 conversations, got %d", len(list))
		}
	})
}

func TestManager_Count(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.Count() != 0 {
		t.Error("should start with 0 conversations")
	}

	if _, err = mgr.GetOrCreate(ctx, "session-1", "/tmp"); err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if mgr.Count() != 1 {
		t.Error("should have 1 conversation")
	}

	if _, err = mgr.GetOrCreate(ctx, "session-2", "/tmp"); err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if mgr.Count() != 2 {
		t.Error("should have 2 conversations")
	}

	if err = mgr.Remove("session-1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if mgr.Count() != 1 {
		t.Error("should have 1 conversation after remove")
	}
}

func TestManager_Close(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if _, err = mgr.GetOrCreate(ctx, "session-1", "/tmp"); err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if _, err = mgr.GetOrCreate(ctx, "session-2", "/tmp"); err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	err = mgr.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if mgr.Count() != 0 {
		t.Error("should have no conversations after close")
	}
}

func TestManager_SetTaskMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	t.Run("set mode", func(t *testing.T) {
		t.Parallel()

		if _, err = mgr.GetOrCreate(ctx, "session-1", "/tmp"); err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

		err = mgr.SetTaskMode("session-1", "compact")
		if err != nil {
			t.Fatalf("SetTaskMode failed: %v", err)
		}

		var mode string
		mode, err = mgr.GetTaskMode("session-1")
		if err != nil {
			t.Fatalf("GetTaskMode failed: %v", err)
		}
		if mode != "compact" {
			t.Errorf("mode mismatch: got %q, want %q", mode, "compact")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		err = mgr.SetTaskMode("non-existent", "compact")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})
}

func TestManager_GetTaskMode(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err = mgr.GetTaskMode("non-existent")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})
}

func newTestManagerWithStorage(t *testing.T) (*Manager, history.Storage) {
	t.Helper()

	tmpDir := t.TempDir()
	histStorage, err := history.NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create history storage: %v", err)
	}

	mgr, err := NewManager(ManagerConfig{
		Factory:        mockFactory(),
		HistoryStorage: histStorage,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	return mgr, histStorage
}

func TestManager_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("save and load", func(t *testing.T) {
		t.Parallel()
		testManagerSaveAndLoad(t, ctx)
	})

	t.Run("load non-existent", func(t *testing.T) {
		t.Parallel()

		mgr, _ := newTestManagerWithStorage(t)
		_, err := mgr.Load(ctx, "non-existent", "/tmp")
		if err == nil {
			t.Fatal("should fail for non-existent session")
		}
	})

	t.Run("save without storage", func(t *testing.T) {
		t.Parallel()

		mgr := newTestManager(t)
		if _, err := mgr.GetOrCreate(ctx, "session-1", "/tmp"); err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}

		if err := mgr.Save("session-1"); err == nil {
			t.Fatal("should fail without storage configured")
		}
	})

	t.Run("load without storage", func(t *testing.T) {
		t.Parallel()

		mgr := newTestManager(t)
		if _, err := mgr.Load(ctx, "session-1", "/tmp"); err == nil {
			t.Fatal("should fail without storage configured")
		}
	})
}

func testManagerSaveAndLoad(t *testing.T, ctx context.Context) {
	t.Helper()

	mgr, histStorage := newTestManagerWithStorage(t)

	conv, err := mgr.GetOrCreate(ctx, "session-save", "/tmp")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if err = conv.history.AddUserMessage("Hello!"); err != nil {
		t.Fatalf("AddUserMessage failed: %v", err)
	}
	if err = conv.history.AddMessage(message.Message{
		Role:    message.RoleAssistant,
		Content: "Hi there!",
	}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if err = mgr.Save("session-save"); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	mgr2, err := NewManager(ManagerConfig{
		Factory:        mockFactory(),
		HistoryStorage: histStorage,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conv2, err := mgr2.Load(ctx, "session-save", "/tmp")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	msgs := conv2.GetHistoryMessages()
	if len(msgs) < 2 {
		t.Errorf("should have at least 2 messages, got %d", len(msgs))
	}
}

func TestManager_Concurrent(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(ManagerConfig{Factory: mockFactory()})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	var wg sync.WaitGroup

	numGoroutines := 100

	// Concurrent GetOrCreate.
	for i := range numGoroutines {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("session-%d", n%10) // 10 unique sessions.

			_, createErr := mgr.GetOrCreate(context.Background(), sessionID, "/tmp")
			if createErr != nil {
				t.Errorf("GetOrCreate failed: %v", createErr)
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
