package curator

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// BenchmarkConvertInsights benchmarks insight to bullet conversion
func BenchmarkConvertInsights(b *testing.B) {
	insights := []*reflector.Insight{
		{
			Content:    "Always validate input parameters before processing",
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
			Evidence:   []string{"validation prevented error"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertInsights(insights)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindDuplicates_EmptyPlaybook benchmarks duplicate detection with empty playbook
func BenchmarkFindDuplicates_EmptyPlaybook(b *testing.B) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	emb, _ := embedder.Embed(ctx, "Always validate input")
	testBullet, _ := bullet.New("Always validate input", bullet.WithEmbedding(emb))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := curator.FindDuplicates(ctx, []*bullet.Bullet{testBullet})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindDuplicates_WithPlaybook benchmarks duplicate detection with populated playbook
func BenchmarkFindDuplicates_WithPlaybook(b *testing.B) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Add 100 bullets to playbook
	for i := 0; i < 100; i++ {
		content := "Test bullet " + string(rune('A'+i%26))
		emb, _ := embedder.Embed(ctx, content)
		b, _ := bullet.New(content, bullet.WithEmbedding(emb))
		pb.Add(ctx, b)
	}

	// New bullet to check
	emb, _ := embedder.Embed(ctx, "Always validate input")
	newBullet, _ := bullet.New("Always validate input", bullet.WithEmbedding(emb))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := curator.FindDuplicates(ctx, []*bullet.Bullet{newBullet})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCurate_SingleInsight benchmarks curating a single insight
func BenchmarkCurate_SingleInsight(b *testing.B) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)

	insights := []*reflector.Insight{
		{
			Content:    "Always validate input parameters before processing",
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req := MergeRequest{Insights: insights}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb := playbook.New(nil, embedder)
		curator := NewCurator(pb, embedder)

		_, err := curator.Curate(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCurate_MultipleInsights benchmarks curating multiple insights
func BenchmarkCurate_MultipleInsights(b *testing.B) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)

	insights := make([]*reflector.Insight, 10)
	for i := 0; i < 10; i++ {
		insights[i] = &reflector.Insight{
			Content:    "Test insight " + string(rune('A'+i)),
			Confidence: 0.8,
			Category:   reflector.CategorySuccessPattern,
		}
	}

	req := MergeRequest{Insights: insights}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb := playbook.New(nil, embedder)
		curator := NewCurator(pb, embedder)

		_, err := curator.Curate(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCosineSimilarity benchmarks the cosine similarity calculation
func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, 384)
	bVec := make([]float32, 384)

	for i := range a {
		a[i] = float32(i) / 384.0
		bVec[i] = float32(i+1) / 384.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cosineSimilarity(a, bVec)
	}
}
