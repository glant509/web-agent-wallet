package logging

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"time"
)

type traceContextKey struct{}

var fallbackIDCounter atomic.Uint64

// TraceContext identifies a distributed trace and the current operation span.
type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	SpanName     string
}

// WithTrace returns a context containing validated trace metadata. Invalid or
// missing IDs are replaced with cryptographically random IDs.
func WithTrace(ctx context.Context, traceID, spanID, parentSpanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	traceID = normalizeID(traceID, 32)
	if traceID == "" {
		traceID = newRandomID(16)
	}
	spanID = normalizeID(spanID, 16)
	if spanID == "" {
		spanID = newRandomID(8)
	}
	parentSpanID = normalizeID(parentSpanID, 16)
	return context.WithValue(ctx, traceContextKey{}, TraceContext{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
	})
}

// StartSpan creates a child span while preserving the current trace. An
// optional name makes the operation easier to identify in logs.
func StartSpan(ctx context.Context, name ...string) context.Context {
	current := TraceFromContext(ctx)
	child := WithTrace(ctx, current.TraceID, "", current.SpanID)
	if len(name) > 0 {
		child = WithSpanName(child, name[0])
	}
	return child
}

// WithSpanName attaches a human-readable operation name to the current span.
func WithSpanName(ctx context.Context, name string) context.Context {
	current := TraceFromContext(ctx)
	if current.TraceID == "" || current.SpanID == "" {
		ctx = WithTrace(ctx, "", "", "")
		current = TraceFromContext(ctx)
	}
	current.SpanName = strings.TrimSpace(name)
	return context.WithValue(ctx, traceContextKey{}, current)
}

// TraceFromContext returns the trace metadata associated with ctx.
func TraceFromContext(ctx context.Context) TraceContext {
	if ctx == nil {
		return TraceContext{}
	}
	trace, _ := ctx.Value(traceContextKey{}).(TraceContext)
	return trace
}

// NewTraceID returns a lowercase 32-character hexadecimal trace ID.
func NewTraceID() string {
	return newRandomID(16)
}

// NewSpanID returns a lowercase 16-character hexadecimal span ID.
func NewSpanID() string {
	return newRandomID(8)
}

// ValidTraceID reports whether value is a non-zero W3C-compatible trace ID.
func ValidTraceID(value string) bool {
	return normalizeID(value, 32) != ""
}

// ValidSpanID reports whether value is a non-zero W3C-compatible span ID.
func ValidSpanID(value string) bool {
	return normalizeID(value, 16) != ""
}

func normalizeID(value string, length int) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != length || strings.Trim(value, "0") == "" {
		return ""
	}
	if _, err := hex.DecodeString(value); err != nil {
		return ""
	}
	return value
}

func newRandomID(size int) string {
	value := make([]byte, size)
	if _, err := crand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	fallback := make([]byte, 16)
	binary.BigEndian.PutUint64(fallback[:8], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(fallback[8:], fallbackIDCounter.Add(1))
	digest := sha256.Sum256(fallback)
	copy(value, digest[:size])
	return hex.EncodeToString(value)
}
