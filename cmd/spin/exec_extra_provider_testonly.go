//go:build e2e_llm_test

package main

import (
	"fmt"
	"os"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/testprovider"
)

// createProviderForExecExtra registers the test-only "test-llm" provider for
// exec mode when built with the e2e_llm_test tag. This keeps the production
// exec mode unaware of the test provider.
//
// When SPIN_TEST_FIXTURE is set, a fixture-based provider is used instead of
// the hardcoded test provider. This enables deterministic, fixture-driven E2E tests.
func createProviderForExecExtra(providerType string) (llm.Provider, bool, error) {
	if providerType == "test-llm" {
		if fixturePath := os.Getenv("SPIN_TEST_FIXTURE"); fixturePath != "" {
			p, err := testprovider.NewFixtureProvider(fixturePath)
			if err != nil {
				return nil, false, fmt.Errorf("load fixture: %w", err)
			}

			return p, true, nil
		}

		return testprovider.NewProvider(), true, nil
	}

	return nil, false, nil
}
