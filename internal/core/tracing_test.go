package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopTracer(t *testing.T) {
	tracer := &NoopTracer{}
	ctx := context.Background()

	// Start a span
	newCtx, span := tracer.Start(ctx, "test-span")
	assert.NotNil(t, span)
	assert.NotNil(t, newCtx)

	// These should all be no-ops
	span.SetAttribute("key", "value")
	span.SetError(assert.AnError)
	span.End()

	// No panics should occur
}

func TestNoopSpan(t *testing.T) {
	span := &NoopSpan{}

	// All methods should be no-ops and not panic
	span.SetAttribute("test", 123)
	span.SetError(assert.AnError)
	span.End()
}

func TestGlobalTracer(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	// Test setting and getting tracer
	customTracer := &NoopTracer{}
	SetGlobalTracer(customTracer)
	assert.Equal(t, customTracer, GetGlobalTracer())

	// Test nil tracer (should not replace)
	SetGlobalTracer(nil)
	assert.Equal(t, customTracer, GetGlobalTracer())
}

func TestStartSpan(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	// Use noop tracer
	SetGlobalTracer(&NoopTracer{})

	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-operation")
	assert.NotNil(t, span)
	assert.NotNil(t, newCtx)

	span.End()
}

func TestStartSpanWithAttributes(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	// Use noop tracer
	SetGlobalTracer(&NoopTracer{})

	ctx := context.Background()
	attrs := []Attribute{
		StringAttr("operation", "test"),
		IntAttr("count", 42),
		BoolAttr("enabled", true),
	}

	newCtx, span := StartSpan(ctx, "test-operation", attrs...)
	assert.NotNil(t, span)
	assert.NotNil(t, newCtx)

	span.End()
}

func TestAttributeCreators(t *testing.T) {
	tests := []struct {
		name     string
		attr     Attribute
		wantKey  string
		wantType interface{}
	}{
		{
			name:     "string attribute",
			attr:     StringAttr("name", "value"),
			wantKey:  "name",
			wantType: "value",
		},
		{
			name:     "int attribute",
			attr:     IntAttr("count", 42),
			wantKey:  "count",
			wantType: 42,
		},
		{
			name:     "bool attribute",
			attr:     BoolAttr("enabled", true),
			wantKey:  "enabled",
			wantType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantKey, tt.attr.Key)
			assert.Equal(t, tt.wantType, tt.attr.Value)
		})
	}
}

// MockTracer for testing tracer integration
type MockTracer struct {
	spans []*MockSpan
}

func (m *MockTracer) Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, Span) {
	span := &MockSpan{
		name:       spanName,
		attributes: make(map[string]interface{}),
		ended:      false,
	}
	for _, attr := range attrs {
		span.attributes[attr.Key] = attr.Value
	}
	m.spans = append(m.spans, span)
	return ctx, span
}

type MockSpan struct {
	name       string
	attributes map[string]interface{}
	err        error
	ended      bool
}

func (m *MockSpan) End() {
	m.ended = true
}

func (m *MockSpan) SetAttribute(key string, value interface{}) {
	m.attributes[key] = value
}

func (m *MockSpan) SetError(err error) {
	m.err = err
}

func TestMockTracer(t *testing.T) {
	tracer := &MockTracer{}
	ctx := context.Background()

	// Create span with attributes
	attrs := []Attribute{
		StringAttr("operation", "test"),
		IntAttr("count", 5),
	}
	newCtx, span := tracer.Start(ctx, "test-span", attrs...)
	assert.NotNil(t, newCtx)

	// Verify span was created
	assert.Len(t, tracer.spans, 1)
	assert.Equal(t, "test-span", tracer.spans[0].name)
	assert.Equal(t, "test", tracer.spans[0].attributes["operation"])
	assert.Equal(t, 5, tracer.spans[0].attributes["count"])

	// Add more attributes
	span.SetAttribute("additional", "value")
	assert.Equal(t, "value", tracer.spans[0].attributes["additional"])

	// Set error
	span.SetError(assert.AnError)
	assert.Equal(t, assert.AnError, tracer.spans[0].err)

	// End span
	span.End()
	assert.True(t, tracer.spans[0].ended)
}

func TestTracerIntegration(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	// Use mock tracer
	mockTracer := &MockTracer{}
	SetGlobalTracer(mockTracer)

	// Simulate traced operation
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "operation",
		StringAttr("type", "test"),
		IntAttr("items", 10),
	)
	defer span.End()
	assert.NotNil(t, newCtx)

	// Do some work
	span.SetAttribute("status", "processing")

	// Verify mock tracer captured everything
	assert.Len(t, mockTracer.spans, 1)
	mockSpan := mockTracer.spans[0]
	assert.Equal(t, "operation", mockSpan.name)
	assert.Equal(t, "test", mockSpan.attributes["type"])
	assert.Equal(t, 10, mockSpan.attributes["items"])
	assert.Equal(t, "processing", mockSpan.attributes["status"])
}

func BenchmarkNoopTracer_Start(b *testing.B) {
	tracer := &NoopTracer{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span")
		span.End()
	}
}

func BenchmarkStartSpan(b *testing.B) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	SetGlobalTracer(&NoopTracer{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := StartSpan(ctx, "benchmark-span")
		span.End()
	}
}

func BenchmarkStartSpanWithAttrs(b *testing.B) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	SetGlobalTracer(&NoopTracer{})
	ctx := context.Background()
	attrs := []Attribute{
		StringAttr("key1", "value1"),
		IntAttr("key2", 42),
		BoolAttr("key3", true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := StartSpan(ctx, "benchmark-span", attrs...)
		span.End()
	}
}
