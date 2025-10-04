package core

import (
	"context"
)

// Tracer interface for distributed tracing.
// This allows optional OpenTelemetry integration without hard dependencies.
type Tracer interface {
	// Start begins a new span with the given name and attributes.
	Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, Span)
}

// Span represents a tracing span.
type Span interface {
	// End completes the span.
	End()

	// SetAttribute sets a key-value attribute on the span.
	SetAttribute(key string, value interface{})

	// SetError marks the span as failed with an error.
	SetError(err error)
}

// Attribute represents a span attribute (key-value pair).
type Attribute struct {
	Key   string
	Value interface{}
}

// NoopTracer is a no-op implementation of Tracer for when tracing is disabled.
type NoopTracer struct{}

// Start returns a no-op span.
func (n *NoopTracer) Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, Span) {
	return ctx, &NoopSpan{}
}

// NoopSpan is a no-op implementation of Span.
type NoopSpan struct{}

// End is a no-op.
func (n *NoopSpan) End() {}

// SetAttribute is a no-op.
func (n *NoopSpan) SetAttribute(key string, value interface{}) {}

// SetError is a no-op.
func (n *NoopSpan) SetError(err error) {}

// Global tracer instance
var globalTracer Tracer = &NoopTracer{}

// SetGlobalTracer sets the global tracer instance.
// This should be called during application initialization if tracing is enabled.
func SetGlobalTracer(t Tracer) {
	if t != nil {
		globalTracer = t
	}
}

// GetGlobalTracer returns the current global tracer.
func GetGlobalTracer() Tracer {
	return globalTracer
}

// StartSpan is a convenience function that starts a span using the global tracer.
func StartSpan(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span) {
	return globalTracer.Start(ctx, name, attrs...)
}

// StringAttr creates a string attribute.
func StringAttr(key, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

// IntAttr creates an integer attribute.
func IntAttr(key string, value int) Attribute {
	return Attribute{Key: key, Value: value}
}

// BoolAttr creates a boolean attribute.
func BoolAttr(key string, value bool) Attribute {
	return Attribute{Key: key, Value: value}
}
