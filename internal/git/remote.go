package git

import (
	"context"
	"fmt"
)

// RemoteURL returns the URL for the specified remote.
// Returns ErrInvalidRemote if the remote doesn't exist.
//
// Example:
//
//	url, err := repo.RemoteURL(ctx, "origin")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Origin URL: %s\n", url)
func (r *Repository) RemoteURL(ctx context.Context, remoteName string) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	remote, err := r.repo.Remote(remoteName)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidRemote, remoteName)
	}

	config := remote.Config()
	if len(config.URLs) == 0 {
		return "", fmt.Errorf("no URLs for remote %s", remoteName)
	}

	return config.URLs[0], nil
}

// ListRemotes returns all configured remotes.
//
// Example:
//
//	remotes, err := repo.ListRemotes(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, remote := range remotes {
//	    fmt.Printf("Remote: %s (%s)\n", remote.Name, remote.URLs[0])
//	}
func (r *Repository) ListRemotes(ctx context.Context) ([]*RemoteInfo, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	remotes, err := r.repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}

	result := make([]*RemoteInfo, 0, len(remotes))
	for _, remote := range remotes {
		config := remote.Config()
		urls := make([]string, 0, len(config.URLs))
		urls = append(urls, config.URLs...)

		result = append(result, &RemoteInfo{
			Name: config.Name,
			URLs: urls,
		})
	}

	return result, nil
}
