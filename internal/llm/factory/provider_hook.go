//go:build !e2e_llm_test

package factory

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// addTestProvider is a no-op in production builds.
func (f *Factory) addTestProvider(_ map[string]func(context.Context, ProviderConfig) (llm.Provider, error)) {
	// No-op.
}
