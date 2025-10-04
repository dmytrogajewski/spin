package core

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OtelTracer wraps OpenTelemetry tracer to implement our Tracer interface.
type OtelTracer struct {
	tracer trace.Tracer
}

// NewOtelTracer creates a new OpenTelemetry-backed tracer.
// The name parameter is used to identify the tracer (e.g., "spin/core").
func NewOtelTracer(name string) *OtelTracer {
	return &OtelTracer{
		tracer: otel.Tracer(name),
	}
}

// Start begins a new span with the given name and attributes.
func (o *OtelTracer) Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, Span) {
	// Convert our attributes to OpenTelemetry attributes
	otelAttrs := make([]attribute.KeyValue, len(attrs))
	for i, attr := range attrs {
		otelAttrs[i] = convertAttribute(attr)
	}

	// Start OpenTelemetry span
	ctx, otelSpan := o.tracer.Start(ctx, spanName, trace.WithAttributes(otelAttrs...))

	// Wrap in our Span interface
	return ctx, &OtelSpan{span: otelSpan}
}

// OtelSpan wraps an OpenTelemetry span to implement our Span interface.
type OtelSpan struct {
	span trace.Span
}

// End completes the span.
func (s *OtelSpan) End() {
	s.span.End()
}

// SetAttribute sets a key-value attribute on the span.
func (s *OtelSpan) SetAttribute(key string, value interface{}) {
	s.span.SetAttributes(convertAttribute(Attribute{Key: key, Value: value}))
}

// SetError marks the span as failed with an error.
func (s *OtelSpan) SetError(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

// convertAttribute converts our Attribute to OpenTelemetry attribute.KeyValue.
func convertAttribute(attr Attribute) attribute.KeyValue {
	switch v := attr.Value.(type) {
	case string:
		return attribute.String(attr.Key, v)
	case int:
		return attribute.Int(attr.Key, v)
	case int64:
		return attribute.Int64(attr.Key, v)
	case float64:
		return attribute.Float64(attr.Key, v)
	case bool:
		return attribute.Bool(attr.Key, v)
	default:
		// Fallback to string representation
		return attribute.String(attr.Key, toString(v))
	}
}

// toString converts any value to string for attribute fallback.
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// InitOtelTracing initializes OpenTelemetry tracing if enabled in config.
// This should be called during application startup.
func InitOtelTracing(cfg *Config) {
	if cfg.EnableTrace {
		tracer := NewOtelTracer("spin/core")
		SetGlobalTracer(tracer)
	}
}
