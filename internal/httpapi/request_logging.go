package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"web3-service-agent/internal/logging"
)

const maxLoggedBodyBytes = 16 * 1024

type cappedCapture struct {
	buffer    bytes.Buffer
	limit     int
	total     int64
	truncated bool
}

func newCappedCapture(limit int) *cappedCapture {
	return &cappedCapture{limit: limit}
}

func (c *cappedCapture) Write(payload []byte) (int, error) {
	c.total += int64(len(payload))
	remaining := c.limit - c.buffer.Len()
	if remaining > 0 {
		written := len(payload)
		if written > remaining {
			written = remaining
		}
		_, _ = c.buffer.Write(payload[:written])
	}
	if c.total > int64(c.limit) {
		c.truncated = true
	}
	return len(payload), nil
}

func (c *cappedCapture) bytes() []byte {
	return c.buffer.Bytes()
}

type captureReadCloser struct {
	io.Reader
	io.Closer
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int64
	capture *cappedCapture
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	if r.capture != nil {
		_, _ = r.capture.Write(payload)
	}
	written, err := r.ResponseWriter.Write(payload)
	r.bytes += int64(written)
	return written, err
}

func (r *statusRecorder) Flush() {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(r.ResponseWriter).Flush()
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID, parentSpanID := incomingTrace(r)
		ctx := logging.WithTrace(r.Context(), traceID, "", parentSpanID)
		ctx = logging.WithSpanName(ctx, r.Method+" "+r.URL.Path)
		r = r.WithContext(ctx)
		trace := logging.TraceFromContext(ctx)

		w.Header().Set("X-Trace-ID", trace.TraceID)
		w.Header().Set("X-Span-ID", trace.SpanID)
		w.Header().Set("traceparent", "00-"+trace.TraceID+"-"+trace.SpanID+"-01")

		requestCapture := newCappedCapture(maxLoggedBodyBytes)
		if r.Body != nil {
			r.Body = captureReadCloser{Reader: io.TeeReader(r.Body, requestCapture), Closer: r.Body}
		}
		responseCapture := newCappedCapture(maxLoggedBodyBytes)
		recorder := &statusRecorder{ResponseWriter: w, capture: responseCapture}

		_ = logging.InfoContext(ctx, "http.request.started", map[string]any{
			"request": requestLogFields(r, nil),
		})

		defer func() {
			panicValue := recover()
			status := recorder.status
			if panicValue != nil {
				status = http.StatusInternalServerError
			} else if status == 0 {
				status = http.StatusOK
			}
			duration := time.Since(start)
			attributes := map[string]any{
				"request":     requestLogFields(r, requestCapture),
				"response":    responseLogFields(recorder, responseCapture, status),
				"duration_ms": float64(duration.Microseconds()) / 1000,
			}
			if panicValue != nil {
				attributes["panic"] = fmt.Sprint(panicValue)
				_ = logging.ErrorContext(ctx, "http.request.completed", attributes)
				panic(panicValue)
			}
			switch {
			case status >= http.StatusInternalServerError:
				_ = logging.ErrorContext(ctx, "http.request.completed", attributes)
			case status >= http.StatusBadRequest:
				_ = logging.WarnContext(ctx, "http.request.completed", attributes)
			default:
				_ = logging.InfoContext(ctx, "http.request.completed", attributes)
			}
		}()

		next.ServeHTTP(recorder, r)
	})
}

func incomingTrace(r *http.Request) (string, string) {
	parts := strings.Split(strings.TrimSpace(r.Header.Get("traceparent")), "-")
	if len(parts) == 4 && len(parts[0]) == 2 && logging.ValidTraceID(parts[1]) && logging.ValidSpanID(parts[2]) {
		return parts[1], parts[2]
	}
	traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID"))
	parentSpanID := strings.TrimSpace(r.Header.Get("X-Span-ID"))
	if !logging.ValidTraceID(traceID) {
		traceID = ""
	}
	if !logging.ValidSpanID(parentSpanID) {
		parentSpanID = ""
	}
	return traceID, parentSpanID
}

func requestLogFields(r *http.Request, capture *cappedCapture) map[string]any {
	fields := map[string]any{
		"method":       r.Method,
		"path":         r.URL.Path,
		"query":        sanitizeQuery(r.URL.Query()),
		"content_type": r.Header.Get("Content-Type"),
		"origin":       r.Header.Get("Origin"),
		"remote_addr":  r.RemoteAddr,
	}
	if capture != nil {
		fields["body"] = sanitizePayload(capture.bytes(), r.Header.Get("Content-Type"), capture.truncated)
		fields["body_bytes"] = capture.total
	}
	return fields
}

func responseLogFields(recorder *statusRecorder, capture *cappedCapture, status int) map[string]any {
	contentType := recorder.Header().Get("Content-Type")
	return map[string]any{
		"status":       status,
		"bytes":        recorder.bytes,
		"content_type": contentType,
		"body":         sanitizePayload(capture.bytes(), contentType, capture.truncated),
	}
}

func sanitizeQuery(values url.Values) map[string]any {
	result := make(map[string]any, len(values))
	for key, entries := range values {
		if sensitiveLogKey(key) {
			result[key] = "[REDACTED]"
			continue
		}
		cleaned := make([]string, 0, len(entries))
		for _, entry := range entries {
			cleaned = append(cleaned, truncateLogString(entry))
		}
		result[key] = cleaned
	}
	return result
}

func sanitizePayload(payload []byte, contentType string, truncated bool) any {
	if len(payload) == 0 {
		return nil
	}
	if strings.Contains(strings.ToLower(contentType), "json") {
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err == nil {
			result := redactLogValue(value)
			if truncated {
				return map[string]any{"value": result, "truncated": true}
			}
			return result
		}
		return map[string]any{
			"omitted":        true,
			"invalid_json":   true,
			"captured_bytes": len(payload),
			"truncated":      truncated,
		}
	}
	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "text/event-stream") {
		return map[string]any{"preview": truncateLogString(string(payload)), "truncated": truncated}
	}
	if strings.HasPrefix(contentType, "text/") || contentType == "" {
		if len(payload) <= 1024 && !truncated {
			return truncateLogString(string(payload))
		}
		return map[string]any{"omitted": true, "captured_bytes": len(payload), "truncated": truncated}
	}
	return map[string]any{"binary": true, "captured_bytes": len(payload), "truncated": truncated}
}

func redactLogValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, entry := range typed {
			if sensitiveLogKey(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = redactLogValue(entry)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, entry := range typed {
			result[index] = redactLogValue(entry)
		}
		return result
	case string:
		return truncateLogString(typed)
	default:
		return value
	}
}

func sensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.TrimSpace(key)))
	switch normalized {
	case "password", "passwordconfirm", "newpassword", "mnemonic", "seed", "privatekey", "secret", "clientsecret", "apikey", "authorization", "accesstoken", "refreshtoken", "rawtransaction":
		return true
	default:
		return false
	}
}

func truncateLogString(value string) string {
	const maxLength = 4096
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength] + "...[TRUNCATED]"
}
