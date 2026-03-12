package curator

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// ConvertInsights transforms insights into bullets.
func ConvertInsights(insights []*reflector.Insight) ([]*bullet.Bullet, error) {
	bullets := make([]*bullet.Bullet, 0, len(insights))

	for _, insight := range insights {
		// Scale confidence (0.0-1.0) to helpful count (0-10).
		helpfulCount := int(insight.Confidence * 10)

		// Build tags from metadata.
		tags := make(map[string]string)
		tags["category"] = string(insight.Category)

		tags["source"] = insight.Source
		if len(insight.Evidence) > 0 {
			tags["evidence"] = joinEvidence(insight.Evidence)
		}

		// Create bullet from insight with tags.
		b, err := bullet.New(insight.Content, bullet.WithTags(tags))
		if err != nil {
			return nil, err
		}

		// Set helpful count based on confidence.
		for range helpfulCount {
			b.IncrementHelpful()
		}

		bullets = append(bullets, b)
	}

	return bullets, nil
}

// joinEvidence concatenates evidence strings with separator.
func joinEvidence(evidence []string) string {
	result := ""

	var resultSb46 strings.Builder
	for i, e := range evidence {
		if i > 0 {
			resultSb46.WriteString("; ")
		}

		resultSb46.WriteString(e)
	}
	result += resultSb46.String()

	return result
}
