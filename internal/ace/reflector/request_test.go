package reflector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

// TestReflectionRequest_Defaults tests default values for request.
func TestReflectionRequest_Defaults(t *testing.T) {
	t.Parallel()

	req := &ReflectionRequest{
		Trajectories: []*generator.Trajectory{
			{ID: "test-1"},
		},
	}

	require.NotNil(t, req)
	assert.Len(t, req.Trajectories, 1)
}

// TestReflectionResponse_Creation tests response creation.
func TestReflectionResponse_Creation(t *testing.T) {
	t.Parallel()

	insights := []*Insight{
		NewInsight("Always validate input parameters before processing them", CategorySuccessPattern),
	}

	resp := &ReflectionResponse{
		Insights:    insights,
		Iterations:  3,
		TotalTokens: 500,
		Duration:    time.Second,
	}

	require.NotNil(t, resp)
	assert.Len(t, resp.Insights, 1)
	assert.Equal(t, 3, resp.Iterations)
	assert.Equal(t, 500, resp.TotalTokens)
	assert.Equal(t, time.Second, resp.Duration)
}
