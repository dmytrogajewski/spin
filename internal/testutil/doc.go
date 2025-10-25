// Package testutil provides common test utilities and helpers for Spin tests.
//
// # Overview
//
// This package reduces test code duplication by providing:
//   - Builder patterns for creating test fixtures
//   - Mock constructors with sensible defaults
//   - Common assertion helpers
//   - Table-driven test utilities
//
// # Usage Example
//
//	func TestMyFeature(t *testing.T) {
//		// Create an agent with default test configuration
//		agent := testutil.NewAgentBuilder(t).Build()
//
//		// Or customize specific components
//		agent := testutil.NewAgentBuilder(t).
//			WithMaxTurns(5).
//			WithTimeout(10 * time.Second).
//			Build()
//
//		// Use test helpers
//		ctx, cancel := testutil.ContextWithTimeout(t)
//		defer cancel()
//
//		result, err := agent.Execute(ctx, req)
//		testutil.RequireNoError(t, err)
//		testutil.AssertNotNil(t, result)
//	}
//
// # Table-Driven Tests
//
// Use the TableTest type for structured table-driven tests:
//
//	tests := []testutil.TableTest{
//		{
//			Name: "successful execution",
//			Setup: func(t *testing.T) interface{} {
//				return testutil.NewAgentBuilder(t).Build()
//			},
//			Execute: func(t *testing.T, fixture interface{}) (interface{}, error) {
//				agent := fixture.(*agent.Agent)
//				return agent.Execute(context.Background(), req)
//			},
//			Assert: func(t *testing.T, result interface{}, err error) {
//				testutil.RequireNoError(t, err)
//				testutil.AssertNotNil(t, result)
//			},
//		},
//	}
//
//	testutil.RunTableTests(t, tests)
//
// # Best Practices
//
//   - Always use t.Helper() in test helpers to get correct line numbers
//   - Use builder patterns for complex test fixtures
//   - Prefer table-driven tests for testing multiple scenarios
//   - Use meaningful test names that describe the scenario
//   - Clean up resources (context cancellation, file cleanup, etc.)
package testutil
