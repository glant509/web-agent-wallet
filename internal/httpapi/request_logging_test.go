package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web3-service-agent/internal/logging"
)

func TestRequestLoggingAddsTraceAndRedactsPayloads(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(t.TempDir(), &output, &output)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	previous := logging.Default()
	logging.SetDefault(logger)
	defer func() {
		logging.SetDefault(previous)
		if err := logger.Close(); err != nil {
			t.Fatalf("close logger: %v", err)
		}
	}()

	handler := withRequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace := logging.TraceFromContext(r.Context())
		if trace.TraceID != "0123456789abcdef0123456789abcdef" || trace.ParentSpanID != "fedcba9876543210" || !logging.ValidSpanID(trace.SpanID) {
			t.Fatalf("unexpected request trace: %+v", trace)
		}
		_, _ = io.ReadAll(r.Body)
		writeJSON(w, http.StatusCreated, map[string]any{
			"status":      "ok",
			"private_key": "must-not-be-logged",
		})
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/test?api_key=hidden&network=base", strings.NewReader(`{"name":"wallet","password":"secret","mnemonic":"one two"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:4312")
	request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-fedcba9876543210-01")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
	if recorder.Header().Get("X-Trace-ID") != "0123456789abcdef0123456789abcdef" || !logging.ValidSpanID(recorder.Header().Get("X-Span-ID")) {
		t.Fatalf("missing response trace headers: %v", recorder.Header())
	}

	entries := decodeJSONLogLines(t, output.String())
	if len(entries) != 2 {
		t.Fatalf("expected start and completion logs, got %d: %s", len(entries), output.String())
	}
	completed := entries[1]
	if completed["message"] != "http.request.completed" || completed["trace_id"] != "0123456789abcdef0123456789abcdef" || completed["parent_span_id"] != "fedcba9876543210" {
		t.Fatalf("unexpected completion log: %+v", completed)
	}
	attributes := completed["attributes"].(map[string]any)
	requestFields := attributes["request"].(map[string]any)
	requestBody := requestFields["body"].(map[string]any)
	if requestFields["origin"] != "http://localhost:4312" {
		t.Fatalf("request origin was not logged: %+v", requestFields)
	}
	if requestBody["password"] != "[REDACTED]" || requestBody["mnemonic"] != "[REDACTED]" || requestBody["name"] != "wallet" {
		t.Fatalf("request body was not safely logged: %+v", requestBody)
	}
	query := requestFields["query"].(map[string]any)
	if query["api_key"] != "[REDACTED]" {
		t.Fatalf("query secret was not redacted: %+v", query)
	}
	responseFields := attributes["response"].(map[string]any)
	responseBody := responseFields["body"].(map[string]any)
	if responseBody["private_key"] != "[REDACTED]" || responseBody["status"] != "ok" {
		t.Fatalf("response body was not safely logged: %+v", responseBody)
	}
	if attributes["duration_ms"].(float64) < 0 {
		t.Fatalf("invalid duration: %+v", attributes["duration_ms"])
	}
}

func TestRequestLoggingPreservesFlusher(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(t.TempDir(), &output, &output)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	previous := logging.Default()
	logging.SetDefault(logger)
	defer func() {
		logging.SetDefault(previous)
		_ = logger.Close()
	}()

	handler := withRequestLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("wrapped response writer lost http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		flusher.Flush()
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/stream", nil))
}

func TestRequestLoggingKeepsSharedTraceAndCreatesRequestSpans(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(t.TempDir(), &output, &output)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	previous := logging.Default()
	logging.SetDefault(logger)
	defer func() {
		logging.SetDefault(previous)
		_ = logger.Close()
	}()

	handler := withRequestLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}))
	const traceID = "b9b9cfdc13383600819fdaaaa187651a"
	parentSpanIDs := []string{"1111111111111111", "2222222222222222"}
	serverSpanIDs := make([]string, 0, len(parentSpanIDs))
	for index, parentSpanID := range parentSpanIDs {
		request := httptest.NewRequest(http.MethodPost, "/v1/wallet/evm/history", strings.NewReader(`{"chain_id":"ethereum"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("traceparent", "00-"+traceID+"-"+parentSpanID+"-01")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", index, recorder.Code)
		}
		if got := recorder.Header().Get("X-Trace-ID"); got != traceID {
			t.Fatalf("request %d: trace changed from %s to %s", index, traceID, got)
		}
		spanID := recorder.Header().Get("X-Span-ID")
		if !logging.ValidSpanID(spanID) {
			t.Fatalf("request %d: invalid server span %q", index, spanID)
		}
		serverSpanIDs = append(serverSpanIDs, spanID)
	}
	if serverSpanIDs[0] == serverSpanIDs[1] {
		t.Fatalf("parallel requests must have distinct server spans: %s", serverSpanIDs[0])
	}

	entries := decodeJSONLogLines(t, output.String())
	if len(entries) != 4 {
		t.Fatalf("expected two log entries per request, got %d: %s", len(entries), output.String())
	}
	seenParents := map[string]bool{}
	seenSpans := map[string]bool{}
	for _, entry := range entries {
		if entry["trace_id"] != traceID {
			t.Fatalf("business trace changed in log entry: %+v", entry)
		}
		parentSpanID, _ := entry["parent_span_id"].(string)
		spanID, _ := entry["span_id"].(string)
		seenParents[parentSpanID] = true
		seenSpans[spanID] = true
	}
	for _, parentSpanID := range parentSpanIDs {
		if !seenParents[parentSpanID] {
			t.Fatalf("missing parent span %s in logs", parentSpanID)
		}
	}
	if len(seenSpans) != 2 {
		t.Fatalf("expected one server span per request, got %v", seenSpans)
	}
}

func decodeJSONLogLines(t *testing.T, value string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(value), "\n")
	entries := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode log line: %v; line=%s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}
