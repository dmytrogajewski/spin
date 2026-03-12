package generator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

func TestTrajectory_Creation(t *testing.T) {
	t.Parallel()

	// Test creating a basic trajectory.
	now := time.Now()

	traj := &Trajectory{
		ID:        "test-id-123",
		Query:     "What is 2+2?",
		CreatedAt: now,
	}

	require.NotNil(t, traj)
	assert.Equal(t, "test-id-123", traj.ID)
	assert.Equal(t, "What is 2+2?", traj.Query)
	assert.Equal(t, now, traj.CreatedAt)
}

func TestTrajectory_WithRetrievedBullets(t *testing.T) {
	t.Parallel()

	// Test trajectory can store retrieved bullets.
	b1, err := bullet.New("Always validate input")
	require.NoError(t, err)

	b2, err := bullet.New("Use context.Context")
	require.NoError(t, err)

	traj := &Trajectory{
		ID:               "test-id-456",
		Query:            "How to write Go code?",
		RetrievedBullets: []*bullet.Bullet{b1, b2},
		CreatedAt:        time.Now(),
	}

	require.NotNil(t, traj)
	assert.Len(t, traj.RetrievedBullets, 2)
	assert.Equal(t, "Always validate input", traj.RetrievedBullets[0].Content)
	assert.Equal(t, "Use context.Context", traj.RetrievedBullets[1].Content)
}

func TestTrajectory_WithOutputAndSuccess(t *testing.T) {
	t.Parallel()

	// Test trajectory can store output and success status.
	traj := &Trajectory{
		ID:        "test-id-789",
		Query:     "Calculate 2+2",
		Output:    "The answer is 4",
		Success:   true,
		CreatedAt: time.Now(),
	}

	require.NotNil(t, traj)
	assert.Equal(t, "The answer is 4", traj.Output)
	assert.True(t, traj.Success)
}

func TestTrajectory_WithSteps(t *testing.T) {
	t.Parallel()

	// Test trajectory can store execution steps.
	step1 := TrajectoryStep{
		StepNumber: 0,
		Type:       "reasoning",
		Content:    "First, I need to add 2+2",
		Timestamp:  time.Now(),
	}

	step2 := TrajectoryStep{
		StepNumber: 1,
		Type:       "reasoning",
		Content:    "The result is 4",
		Timestamp:  time.Now(),
	}

	traj := &Trajectory{
		ID:        "test-id-999",
		Query:     "What is 2+2?",
		Steps:     []TrajectoryStep{step1, step2},
		Output:    "4",
		Success:   true,
		CreatedAt: time.Now(),
	}

	require.NotNil(t, traj)
	assert.Len(t, traj.Steps, 2)
	assert.Equal(t, 0, traj.Steps[0].StepNumber)
	assert.Equal(t, "reasoning", traj.Steps[0].Type)
	assert.Equal(t, "First, I need to add 2+2", traj.Steps[0].Content)
	assert.Equal(t, 1, traj.Steps[1].StepNumber)
}

func TestTrajectory_WithFeedback(t *testing.T) {
	t.Parallel()

	// Test trajectory can store bullet feedback.
	fb := &BulletFeedback{
		HelpfulBullets: []string{"bullet-1", "bullet-2"},
		HarmfulBullets: []string{"bullet-3"},
		Explanation:    "Bullets 1 and 2 were helpful",
	}

	traj := &Trajectory{
		ID:             "test-with-feedback",
		Query:          "Test query",
		BulletFeedback: fb,
		CreatedAt:      time.Now(),
	}

	require.NotNil(t, traj)
	require.NotNil(t, traj.BulletFeedback)
	assert.Len(t, traj.BulletFeedback.HelpfulBullets, 2)
	assert.Len(t, traj.BulletFeedback.HarmfulBullets, 1)
	assert.Equal(t, "Bullets 1 and 2 were helpful", traj.BulletFeedback.Explanation)
}

func TestTrajectory_WithMetadata(t *testing.T) {
	t.Parallel()

	// Test trajectory can store metadata.
	metadata := TrajectoryMetadata{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
		TotalTokens: 500,
		Duration:    2 * time.Second,
		Turns:       3,
	}

	traj := &Trajectory{
		ID:        "test-with-metadata",
		Query:     "Test query",
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	require.NotNil(t, traj)
	assert.Equal(t, "gpt-4", traj.Metadata.Model)
	assert.Equal(t, 0.7, traj.Metadata.Temperature)
	assert.Equal(t, 1000, traj.Metadata.MaxTokens)
	assert.Equal(t, 500, traj.Metadata.TotalTokens)
	assert.Equal(t, 2*time.Second, traj.Metadata.Duration)
	assert.Equal(t, 3, traj.Metadata.Turns)
}

func TestTrajectory_Complete(t *testing.T) {
	t.Parallel()

	// Test complete trajectory with all fields.
	b1, err := bullet.New("Test bullet")
	require.NoError(t, err)

	feedback := &BulletFeedback{
		HelpfulBullets: []string{b1.ID},
		HarmfulBullets: []string{},
	}

	metadata := TrajectoryMetadata{
		Model:       "gpt-4",
		Temperature: 0.7,
		TotalTokens: 100,
		Turns:       1,
	}

	step := TrajectoryStep{
		StepNumber: 0,
		Type:       "reasoning",
		Content:    "Solving...",
		Timestamp:  time.Now(),
	}

	traj := &Trajectory{
		ID:               "complete-traj",
		Query:            "Complete test",
		RetrievedBullets: []*bullet.Bullet{b1},
		Steps:            []TrajectoryStep{step},
		Output:           "Solution",
		Success:          true,
		BulletFeedback:   feedback,
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}

	require.NotNil(t, traj)
	assert.Equal(t, "complete-traj", traj.ID)
	assert.Equal(t, "Complete test", traj.Query)
	assert.Len(t, traj.RetrievedBullets, 1)
	assert.Len(t, traj.Steps, 1)
	assert.Equal(t, "Solution", traj.Output)
	assert.True(t, traj.Success)
	assert.NotNil(t, traj.BulletFeedback)
	assert.Equal(t, "gpt-4", traj.Metadata.Model)
}
