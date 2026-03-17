package retrieval

import "context"

// bulletSourceName is the source identifier for ACE bullet retrieval.
const bulletSourceName = "ace_bullets"

// BulletSource extracts active ACE bullets from the trajectory context.
type BulletSource struct{}

// NewBulletSource creates a BulletSource.
func NewBulletSource() *BulletSource {
	return &BulletSource{}
}

// Name returns the source identifier.
func (b *BulletSource) Name() string { return bulletSourceName }

// Retrieve extracts active bullets from the trajectory context.
// Returns empty fragments if no trajectory context is available.
func (b *BulletSource) Retrieve(
	_ context.Context, req Request,
) ([]Fragment, error) {
	if req.TrajectoryCtx == nil {
		return nil, nil
	}

	bullets := req.TrajectoryCtx.GetActiveBullets()
	if len(bullets) == 0 {
		return nil, nil
	}

	fragments := make([]Fragment, 0, len(bullets))

	for _, blt := range bullets {
		fragments = append(fragments, Fragment{
			Source:  bulletSourceName,
			Content: blt.Content,
		})
	}

	return fragments, nil
}
