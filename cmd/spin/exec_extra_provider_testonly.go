//go:build e2e_llm_test

package main

import (
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/testprovider"
)

// createProviderForExecExtra registers the test-only "test-llm" provider for
// exec mode when built with the e2e_llm_test tag. This keeps the production
// exec mode unaware of the test provider.
func createProviderForExecExtra(providerType string) (llm.Provider, bool, error) {
	if providerType == "test-llm" {
		return testprovider.NewProvider(), true, nil
	}
	return nil, false, nil
}
