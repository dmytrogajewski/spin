//go:build e2e_llm_test

package factory

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/testprovider"
)

// addTestProvider adds the test provider in E2E test builds.
func (f *Factory) addTestProvider(providers map[string]func(context.Context, ProviderConfig) (llm.Provider, error)) {
	providers["test-llm"] = f.newTestProvider
}

// newTestProvider creates a new test provider.
func (f *Factory) newTestProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	return testprovider.NewProvider(), nil
}
