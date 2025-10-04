package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOtelTracer(t *testing.T) {
	tracer := NewOtelTracer("test/tracer")
	assert.NotNil(t, tracer)
	assert.NotNil(t, tracer.tracer)
}

func TestOtelTracer_Start(t *testing.T) {
	tracer := NewOtelTracer("test/tracer")
	ctx := context.Background()

	// Start a span
	ctx, span := tracer.Start(ctx, "test-operation",
		StringAttr("key1", "value1"),
		IntAttr("key2", 42),
	)
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	// Type assertion to check it's an OtelSpan
	otelSpan, ok := span.(*OtelSpan)
	assert.True(t, ok)
	assert.NotNil(t, otelSpan.span)

	span.End()
}

func TestOtelSpan_SetAttribute(t *testing.T) {
	tracer := NewOtelTracer("test/tracer")
	ctx := context.Background()

	newCtx, span := tracer.Start(ctx, "test-operation")
	defer span.End()
	assert.NotNil(t, newCtx)

	// Set various attribute types
	span.SetAttribute("string_attr", "value")
	span.SetAttribute("int_attr", 123)
	span.SetAttribute("bool_attr", true)
	span.SetAttribute("float_attr", 3.14)

	// Should not panic
}

func TestOtelSpan_SetError(t *testing.T) {
	tracer := NewOtelTracer("test/tracer")
	ctx := context.Background()

	newCtx, span := tracer.Start(ctx, "test-operation")
	defer span.End()
	assert.NotNil(t, newCtx)

	// Set an error
	testErr := errors.New("test error")
	span.SetError(testErr)

	// Should not panic
}

func TestConvertAttribute(t *testing.T) {
	tests := []struct {
		name string
		attr Attribute
	}{
		{
			name: "string attribute",
			attr: StringAttr("key", "value"),
		},
		{
			name: "int attribute",
			attr: IntAttr("key", 42),
		},
		{
			name: "bool attribute",
			attr: BoolAttr("key", true),
		},
		{
			name: "int64 attribute",
			attr: Attribute{Key: "key", Value: int64(123)},
		},
		{
			name: "float64 attribute",
			attr: Attribute{Key: "key", Value: float64(3.14)},
		},
		{
			name: "unsupported type fallback",
			attr: Attribute{Key: "key", Value: struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			kv := convertAttribute(tt.attr)
			assert.NotEmpty(t, kv.Key)
		})
	}
}

func TestInitOtelTracing(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	tests := []struct {
		name       string
		cfg        *Config
		expectOtel bool
	}{
		{
			name: "tracing enabled",
			cfg: &Config{
				EnableTrace: true,
			},
			expectOtel: true,
		},
		{
			name: "tracing disabled",
			cfg: &Config{
				EnableTrace: false,
			},
			expectOtel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to noop before each test
			SetGlobalTracer(&NoopTracer{})

			// Initialize
			InitOtelTracing(tt.cfg)

			// Check if tracer was set
			tracer := GetGlobalTracer()
			if tt.expectOtel {
				_, ok := tracer.(*OtelTracer)
				assert.True(t, ok, "Expected OtelTracer when tracing enabled")
			} else {
				_, ok := tracer.(*NoopTracer)
				assert.True(t, ok, "Expected NoopTracer when tracing disabled")
			}
		})
	}
}

func TestOtelTracer_Integration(t *testing.T) {
	// Save original tracer
	original := GetGlobalTracer()
	defer SetGlobalTracer(original)

	// Set up OTel tracer
	tracer := NewOtelTracer("test/integration")
	SetGlobalTracer(tracer)

	ctx := context.Background()

	// Simulate nested spans (like Agent -> Executor)
	ctx, parentSpan := StartSpan(ctx, "ParentOperation",
		StringAttr("operation", "test"),
	)
	defer parentSpan.End()

	// Child span
	ctx, childSpan := StartSpan(ctx, "ChildOperation",
		IntAttr("count", 5),
	)
	assert.NotNil(t, ctx)

	// Set attributes and error
	childSpan.SetAttribute("status", "processing")
	childSpan.SetError(errors.New("test error"))

	childSpan.End()

	// Parent span continues
	parentSpan.SetAttribute("child_complete", true)

	// Should not panic, spans properly nested
}

func BenchmarkOtelTracer_Start(b *testing.B) {
	tracer := NewOtelTracer("bench/tracer")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span")
		span.End()
	}
}

func BenchmarkOtelTracer_WithAttributes(b *testing.B) {
	tracer := NewOtelTracer("bench/tracer")
	ctx := context.Background()
	attrs := []Attribute{
		StringAttr("key1", "value1"),
		IntAttr("key2", 42),
		BoolAttr("key3", true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "benchmark-span", attrs...)
		span.End()
	}
}

func BenchmarkOtelSpan_SetAttribute(b *testing.B) {
	tracer := NewOtelTracer("bench/tracer")
	_, span := tracer.Start(context.Background(), "benchmark-span")
	defer span.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span.SetAttribute("iteration", i)
	}
}
