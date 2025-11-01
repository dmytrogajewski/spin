package reflector

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// BenchmarkReflector_Reflect_SingleTrajectory benchmarks single trajectory reflection
func BenchmarkReflector_Reflect_SingleTrajectory(b *testing.B) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always validate input parameters before processing them in Go",
			"evidence": ["validation prevented error"],
			"confidence": 0.9,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)
	ctx := context.Background()

	traj := &generator.Trajectory{
		ID:      "bench-1",
		Query:   "How to validate inputs?",
		Output:  "Always validate parameters",
		Success: true,
	}

	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reflector.Reflect(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReflector_Reflect_MultipleTrajectories benchmarks batch trajectory analysis
func BenchmarkReflector_Reflect_MultipleTrajectories(b *testing.B) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use defer for resource cleanup in Go to prevent leaks",
			"evidence": ["Multiple trajectories used defer successfully"],
			"confidence": 0.95,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)
	ctx := context.Background()

	trajectories := make([]*generator.Trajectory, 10)
	for i := 0; i < 10; i++ {
		trajectories[i] = &generator.Trajectory{
			ID:      "bench-multi",
			Query:   "Resource management",
			Output:  "Use defer for cleanup",
			Success: true,
		}
	}

	req := ReflectionRequest{
		Trajectories: trajectories,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reflector.Reflect(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReflector_RefineInsights benchmarks insight refinement
func BenchmarkReflector_RefineInsights(b *testing.B) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is for error type checking in Go to avoid interface comparison issues",
			"evidence": ["errors.Is prevents comparison bugs"],
			"confidence": 0.95,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)
	ctx := context.Background()

	insights := []*Insight{
		{
			Content:    "Always use errors.Is for error type checking in Go",
			Confidence: 0.8,
			Category:   CategorySuccessPattern,
			Evidence:   []string{"errors.Is works well"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reflector.RefineInsights(ctx, insights, 3)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInsightValidator_Validate benchmarks insight validation
func BenchmarkInsightValidator_Validate(b *testing.B) {
	validator := NewInsightValidator()

	insight := &Insight{
		Content:    "Always validate input parameters before processing them",
		Confidence: 0.85,
		Category:   CategorySuccessPattern,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(insight)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInsightValidator_FilterByQuality benchmarks quality filtering
func BenchmarkInsightValidator_FilterByQuality(b *testing.B) {
	validator := NewInsightValidator()

	insights := make([]*Insight, 100)
	for i := 0; i < 100; i++ {
		insights[i] = &Insight{
			Content:    "Always validate input parameters before processing them",
			Confidence: float64(i) / 100.0,
			Category:   CategorySuccessPattern,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.FilterByQuality(insights, 0.5)
	}
}

// BenchmarkPromptBuilder_BuildSingleTrajectory benchmarks prompt building
func BenchmarkPromptBuilder_BuildSingleTrajectory(b *testing.B) {
	builder := NewPromptBuilder()

	traj := &generator.Trajectory{
		ID:      "bench-1",
		Query:   "How to handle errors in Go?",
		Output:  "Use errors.Is for type checking",
		Success: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.BuildSingleTrajectory(traj)
	}
}

// BenchmarkPromptBuilder_BuildBatchTrajectory benchmarks batch prompt building
func BenchmarkPromptBuilder_BuildBatchTrajectory(b *testing.B) {
	builder := NewPromptBuilder()

	trajectories := make([]*generator.Trajectory, 10)
	for i := 0; i < 10; i++ {
		trajectories[i] = &generator.Trajectory{
			ID:      "bench-multi",
			Query:   "Resource management patterns",
			Output:  "Use defer for cleanup",
			Success: true,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.BuildBatchTrajectory(trajectories)
	}
}

// BenchmarkPromptBuilder_BuildRefinementPrompt benchmarks refinement prompt building
func BenchmarkPromptBuilder_BuildRefinementPrompt(b *testing.B) {
	builder := NewPromptBuilder()

	insights := []*Insight{
		{
			Content:    "Always validate input parameters before processing them",
			Confidence: 0.8,
			Category:   CategorySuccessPattern,
			Evidence:   []string{"validation prevented error"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.BuildRefinementPrompt(insights)
	}
}
