//go:build !e2e_llm_test

package main

import "github.com/dmytrogajewski/spin/internal/llm"

// createProviderForACPExtra allows build-tagged extensions to provide additional
// providers for the ACP server. The default implementation returns no match.
func createProviderForACPExtra(_, _, _, _ string) (llm.Provider, bool, error) {
	return nil, false, nil
}
