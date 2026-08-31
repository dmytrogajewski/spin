package child

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

// DialRemote fetches a remote Agent Card over HTTPS when cardURL is on allowlist.
func DialRemote(
	ctx context.Context,
	cardURL string,
	allowlist []string,
	opts ...a2a.HTTPOption,
) (*a2a.HTTPClient, error) {
	return a2a.DialHTTP(ctx, cardURL, a2a.Allowlist(allowlist), opts...)
}
