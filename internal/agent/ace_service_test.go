package agent

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/feedback"
)

// Helper function for substring checking in tests.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Test ACEService creation with valid config.
func TestNewACEService_Enabled(t *testing.T) {
	t.Parallel()
	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: "/tmp/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	svc, err := NewACEService(cfg, "/tmp/workdir", nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	if svc == nil {
		t.Fatal("NewACEService() returned nil service")
	}
}

// Test ACEService creation with disabled config returns no-op.
func TestNewACEService_Disabled(t *testing.T) {
	t.Parallel()
	cfg := &ACEConfig{
		Enabled: false,
	}

	svc, err := NewACEService(cfg, "/tmp/workdir", nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	if svc == nil {
		t.Fatal("NewACEService() returned nil service")
	}

	// No-op service should return empty results without errors.
	ctx := context.Background()

	bullets, err := svc.Retrieve(ctx, "test query")
	if err != nil {
		t.Errorf("Retrieve() on disabled service error = %v", err)
	}

	if len(bullets) != 0 {
		t.Errorf("Retrieve() on disabled service returned %d bullets, want 0", len(bullets))
	}
}

// Test SavePlaybook saves to configured path.
func TestACEService_SavePlaybook(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	playbookPath := tmpDir + "/test-playbook.json"

	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: playbookPath,
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	svc, err := NewACEService(cfg, tmpDir, nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	// Save playbook.
	err = svc.SavePlaybook()
	if err != nil {
		t.Errorf("SavePlaybook() error = %v", err)
	}

	// Verify file exists.
	_, err = os.Stat(playbookPath)
	if os.IsNotExist(err) {
		t.Errorf("Playbook file not created at %s", playbookPath)
	}
}

// Test BuildPrompt creates system prompt with bullets.
func TestACEService_BuildPrompt(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
		ItemizedLearning: ACEItemizedLearningConfig{
			Enabled:       true,
			ParseFeedback: true,
			UpdateAsync:   true,
		},
	}

	svc, err := NewACEService(cfg, tmpDir, nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	ctx := context.Background()

	// Add a test bullet to playbook.
	testBullet := &bullet.Bullet{
		ID:      "test-1",
		Content: "Use sync.RWMutex for concurrent maps",
	}
	err = svc.playbook.Add(ctx, testBullet)
	if err != nil {
		t.Fatalf("Failed to add bullet: %v", err)
	}

	systemPrompt := "You are a Go coding expert."

	prompt, err := svc.BuildPrompt(ctx, systemPrompt, []*bullet.Bullet{testBullet})
	if err != nil {
		t.Errorf("BuildPrompt() error = %v", err)
	}

	// Verify prompt contains system prompt.
	if !contains(prompt, systemPrompt) {
		t.Errorf("BuildPrompt() missing system prompt")
	}

	// Verify prompt contains bullet content.
	if !contains(prompt, testBullet.Content) {
		t.Errorf("BuildPrompt() missing bullet content")
	}

	// Verify prompt contains ItemizedLearning instructions when enabled.
	if !contains(prompt, "HELPFUL") {
		t.Errorf("BuildPrompt() missing ItemizedLearning instructions")
	}
}

// Test BuildPrompt when disabled returns base prompt.
func TestACEService_BuildPrompt_Disabled(t *testing.T) {
	t.Parallel()
	cfg := &ACEConfig{
		Enabled: false,
	}

	svc, err := NewACEService(cfg, "/tmp", nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	ctx := context.Background()
	systemPrompt := "You are helpful."

	prompt, err := svc.BuildPrompt(ctx, systemPrompt, nil)
	if err != nil {
		t.Errorf("BuildPrompt() error = %v", err)
	}

	// When disabled, should return original system prompt unchanged.
	if prompt != systemPrompt {
		t.Errorf("BuildPrompt() on disabled service = %q, want %q", prompt, systemPrompt)
	}
}

// Test ParseFeedback extracts HELPFUL/HARMFUL markers.
func TestACEService_ParseFeedback(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	svc, err := NewACEService(cfg, tmpDir, nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	// LLM response with feedback markers.
	response := `Here's the solution using sync.RWMutex as suggested.

HELPFUL: [B0, B2]
HARMFUL: [B1]
EXPLANATION: B0 and B2 were directly applicable, B1 suggested the wrong approach.`

	feedback, err := svc.ParseFeedback(response)
	if err != nil {
		t.Errorf("ParseFeedback() error = %v", err)
	}

	if feedback == nil {
		t.Fatal("ParseFeedback() returned nil feedback")
	}

	// Verify helpful markers.
	if len(feedback.HelpfulBullets) != 2 {
		t.Errorf("ParseFeedback() helpful bullets = %d, want 2", len(feedback.HelpfulBullets))
	}

	// Verify harmful markers.
	if len(feedback.HarmfulBullets) != 1 {
		t.Errorf("ParseFeedback() harmful bullets = %d, want 1", len(feedback.HarmfulBullets))
	}

	// Verify explanation.
	if !contains(feedback.Explanation, "directly applicable") {
		t.Errorf("ParseFeedback() missing explanation")
	}
}

// Test ParseFeedback when disabled returns nil.
func TestACEService_ParseFeedback_Disabled(t *testing.T) {
	t.Parallel()
	cfg := &ACEConfig{
		Enabled: false,
	}

	svc, err := NewACEService(cfg, "/tmp", nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	feedback, err := svc.ParseFeedback("HELPFUL: [B0]")
	if !errors.Is(err, ErrACEDisabled) {
		t.Errorf("ParseFeedback() on disabled service should return ErrACEDisabled, got %v", err)
	}

	// When disabled, should return nil feedback.
	if feedback != nil {
		t.Errorf("ParseFeedback() on disabled service should return nil feedback")
	}
}

// Test UpdateBullets increments counters based on feedback.
func TestACEService_UpdateBullets(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	svc, err := NewACEService(cfg, tmpDir, nil, "", 0)
	if err != nil {
		t.Fatalf("NewACEService() error = %v", err)
	}

	ctx := context.Background()

	// Add test bullets with indexed IDs (B0, B1, B2).
	bullets := []*bullet.Bullet{
		{ID: "test-b0", Content: "Use RWMutex"},
		{ID: "test-b1", Content: "Use channels"},
		{ID: "test-b2", Content: "Use atomic"},
	}

	for _, b := range bullets {
		err = svc.playbook.Add(ctx, b)
		if err != nil {
			t.Fatalf("Failed to add bullet: %v", err)
		}
	}

	// Create feedback with bullet markers (B0, B1, B2 correspond to indices).
	fb := &feedback.BulletFeedback{
		HelpfulBullets: []string{"B0", "B2"}, // First and third bullets helpful.
		HarmfulBullets: []string{"B1"},       // Second bullet harmful.
	}

	// Update bullets.
	err = svc.UpdateBullets(ctx, bullets, fb)
	if err != nil {
		t.Errorf("UpdateBullets() error = %v", err)
	}

	// Verify counters were updated
	// Note: We need to retrieve bullets to check counters.
	updated0, _ := svc.playbook.Get("test-b0")
	if updated0.HelpfulCount != 1 {
		t.Errorf("Bullet B0 helpful count = %d, want 1", updated0.HelpfulCount)
	}

	updated1, _ := svc.playbook.Get("test-b1")
	if updated1.HarmfulCount != 1 {
		t.Errorf("Bullet B1 harmful count = %d, want 1", updated1.HarmfulCount)
	}

	updated2, _ := svc.playbook.Get("test-b2")
	if updated2.HelpfulCount != 1 {
		t.Errorf("Bullet B2 helpful count = %d, want 1", updated2.HelpfulCount)
	}
}
