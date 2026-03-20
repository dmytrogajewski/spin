package acp

// Journey: specs/journeys/JOURNEY-R2.1-acp-session-list.md.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/storage"
)

// TestUnstableListSessions_NoStorage tests error when storage is not configured.
func TestUnstableListSessions_NoStorage(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	_, err = acpAgent.UnstableListSessions(context.Background(), acp.UnstableListSessionsRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionPersistenceNotAvailable)
}

// TestUnstableListSessions_EmptyStorage tests listing with no saved sessions.
func TestUnstableListSessions_EmptyStorage(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	resp, err := acpAgent.UnstableListSessions(context.Background(), acp.UnstableListSessionsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Sessions)
	assert.Nil(t, resp.NextCursor)
}

// TestUnstableListSessions_MultipleSessions tests listing multiple saved sessions.
func TestUnstableListSessions_MultipleSessions(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	sessions := createTestSessions(ctx, t, store, 3, "/home/user/project")

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, len(sessions))
	assert.Nil(t, resp.NextCursor)
}

// TestUnstableListSessions_CwdFilter tests filtering by working directory.
func TestUnstableListSessions_CwdFilter(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	createTestSessions(ctx, t, store, 2, "/home/user/project-a")
	createTestSessions(ctx, t, store, 1, "/home/user/project-b")

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	filterCwd := "/home/user/project-a"
	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{
		Cwd: &filterCwd,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, 2)

	for _, si := range resp.Sessions {
		assert.Equal(t, filterCwd, si.Cwd)
	}
}

// TestUnstableListSessions_FieldMapping tests that session fields are correctly mapped.
func TestUnstableListSessions_FieldMapping(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	sess := session.NewSession("/home/user/work")
	sess.Metadata.Title = "Test Session"

	require.NoError(t, store.Save(ctx, sess.ID, *sess))

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Sessions, 1)

	si := resp.Sessions[0]
	assert.Equal(t, acp.SessionId(sess.ID), si.SessionId)
	assert.Equal(t, "/home/user/work", si.Cwd)
	require.NotNil(t, si.Title)
	assert.Equal(t, "Test Session", *si.Title)
	require.NotNil(t, si.UpdatedAt)
}

// TestUnstableListSessions_Pagination tests cursor-based pagination.
func TestUnstableListSessions_Pagination(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	totalSessions := listSessionsPageSize + 10
	createTestSessions(ctx, t, store, totalSessions, "/home/user/project")

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	// First page.
	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, listSessionsPageSize)
	require.NotNil(t, resp.NextCursor, "should have next cursor when more sessions exist")

	// Second page.
	resp2, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{
		Cursor: resp.NextCursor,
	})
	require.NoError(t, err)
	assert.Len(t, resp2.Sessions, 10)
	assert.Nil(t, resp2.NextCursor, "no more pages")
}

// TestUnstableListSessions_EmptyTitle tests that empty title maps to nil.
func TestUnstableListSessions_EmptyTitle(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	sess := session.NewSession("/tmp/test")

	require.NoError(t, store.Save(ctx, sess.ID, *sess))

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Sessions, 1)
	assert.Nil(t, resp.Sessions[0].Title, "empty title should map to nil")
}

// TestBuildAgentCapabilities_WithStorage tests that list capability is advertised.
func TestBuildAgentCapabilities_WithStorage(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	caps := acpAgent.buildAgentCapabilities()
	assert.NotNil(t, caps.SessionCapabilities.List, "should advertise list capability when storage is available")
}

// TestBuildAgentCapabilities_WithoutStorage tests that list capability is not advertised.
func TestBuildAgentCapabilities_WithoutStorage(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	caps := acpAgent.buildAgentCapabilities()
	assert.Nil(t, caps.SessionCapabilities.List, "should not advertise list capability without storage")
}

// createTestSessions is a helper that creates and persists test sessions.
func createTestSessions(
	ctx context.Context, t *testing.T, store session.Storage, count int, cwd string,
) []*session.Session {
	t.Helper()

	sessions := make([]*session.Session, 0, count)

	for range count {
		sess := session.NewSession(cwd)
		sess.UpdatedAt = time.Now()
		require.NoError(t, store.Save(ctx, sess.ID, *sess))

		sessions = append(sessions, sess)
	}

	return sessions
}

// TestUnstableListSessions_PaginationWithCwdFilter tests pagination combined with filtering.
func TestUnstableListSessions_PaginationWithCwdFilter(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()

	// Create more than one page of matching sessions.
	matchCount := listSessionsPageSize + 5
	createTestSessions(ctx, t, store, matchCount, "/matching")
	createTestSessions(ctx, t, store, 3, "/other")

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	filterCwd := "/matching"

	// First page — should get pageSize items.
	resp, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{
		Cwd: &filterCwd,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, listSessionsPageSize)
	require.NotNil(t, resp.NextCursor)

	// Second page — should get remaining.
	resp2, err := acpAgent.UnstableListSessions(ctx, acp.UnstableListSessionsRequest{
		Cwd:    &filterCwd,
		Cursor: resp.NextCursor,
	})
	require.NoError(t, err)

	totalReturned := len(resp.Sessions) + len(resp2.Sessions)
	assert.Equal(t, matchCount, totalReturned)
}

// TestUnstableListSessions_InvalidCursor tests graceful handling of invalid cursor.
func TestUnstableListSessions_InvalidCursor(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	invalidCursor := "not-a-number"
	resp, err := acpAgent.UnstableListSessions(context.Background(), acp.UnstableListSessionsRequest{
		Cursor: &invalidCursor,
	})
	require.NoError(t, err, "invalid cursor should be treated as start")
	assert.Empty(t, resp.Sessions)
}

// paginateAll is a helper that fetches all pages via cursor pagination.
func paginateAll(
	t *testing.T, acpAgent *SpinACPAgent, req acp.UnstableListSessionsRequest,
) []acp.UnstableSessionInfo {
	t.Helper()

	var all []acp.UnstableSessionInfo

	cursor := req.Cursor

	for {
		req.Cursor = cursor

		resp, err := acpAgent.UnstableListSessions(context.Background(), req)
		require.NoError(t, err)

		all = append(all, resp.Sessions...)

		if resp.NextCursor == nil {
			break
		}

		cursor = resp.NextCursor
	}

	return all
}

// TestUnstableListSessions_PaginateAll verifies all sessions are reachable via pagination.
func TestUnstableListSessions_PaginateAll(t *testing.T) {
	t.Parallel()

	store, storeErr := storage.NewFileStore[session.Session](storage.FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".json",
	})
	require.NoError(t, storeErr)

	ctx := context.Background()
	totalCount := listSessionsPageSize*2 + 7
	createTestSessions(ctx, t, store, totalCount, "/test")

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		store,
	)
	require.NoError(t, err)

	all := paginateAll(t, acpAgent, acp.UnstableListSessionsRequest{})
	assert.Len(t, all, totalCount)

	// Verify all IDs are unique.
	seen := make(map[acp.SessionId]bool, len(all))
	for _, si := range all {
		assert.False(t, seen[si.SessionId], "duplicate session ID: %s", si.SessionId)

		seen[si.SessionId] = true
	}
}
