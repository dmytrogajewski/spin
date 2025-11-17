//go:build e2e_llm_test

package main

import (
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/testprovider"
)

// createProviderForACPExtra registers the test-only "test-llm" provider for
// ACP when built with the e2e_llm_test tag. This keeps the production ACP
// server unaware of the test provider.
func createProviderForACPExtra(providerType, baseURL, model, apiKey string) (llm.Provider, bool, error) {
	if providerType == "test-llm" {
		return testprovider.NewProvider(), true, nil
	}
	return nil, false, nil
}


