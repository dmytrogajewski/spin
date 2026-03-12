//go:build !e2e_llm_test

package main

import (
	"github.com/dmytrogajewski/spin/internal/llm"
)

// createProviderForExecExtra allows build-tagged extensions to provide additional
// providers for exec mode. This stub returns nil when not built with e2e_llm_test.
func createProviderForExecExtra(_ string) (llm.Provider, bool, error) {
	return nil, false, nil
}
