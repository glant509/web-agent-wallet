package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewCreatesLevelFilesAndWritesOutputs(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger, err := New(dir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatalf("close logger: %v", err)
		}
	}()

	logger.Debugf("debug message")
	logger.Infof("info message")
	logger.Warnf("warn message")
	logger.Errorf("error message")

	assertFileContains(t, filepath.Join(dir, "logs", "debug.log"), "debug message")
	assertFileContains(t, filepath.Join(dir, "logs", "info.log"), "info message")
	assertFileContains(t, filepath.Join(dir, "logs", "warn.log"), "warn message")
	assertFileContains(t, filepath.Join(dir, "logs", "error.log"), "error message")

	if !strings.Contains(stdout.String(), `"level":"DEBUG"`) || !strings.Contains(stdout.String(), "debug message") {
		t.Fatalf("stdout missing debug message: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"level":"INFO"`) || !strings.Contains(stdout.String(), "info message") {
		t.Fatalf("stdout missing info message: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), `"level":"WARN"`) || !strings.Contains(stderr.String(), "warn message") {
		t.Fatalf("stderr missing warn message: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), `"level":"ERROR"`) || !strings.Contains(stderr.String(), "error message") {
		t.Fatalf("stderr missing error message: %s", stderr.String())
	}
	assertLogMetadata(t, stdout.String(), "logging_test.go", "logging.TestNewCreatesLevelFilesAndWritesOutputs")
}

func TestEveryLineContainsMetadata(t *testing.T) {
	var output bytes.Buffer
	logger := newConsoleLogger(&output, io.Discard)

	logger.Infof("first \"line\"\nsecond line")

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two log lines, got %d: %q", len(lines), output.String())
	}
	for _, line := range lines {
		assertLogMetadata(t, line, "logging_test.go", "logging.TestEveryLineContainsMetadata")
	}
}

func TestContextWriteHonorsDeadline(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	logger := newConsoleLogger(writer, io.Discard)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := logger.InfofContext(ctx, "blocked message")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	select {
	case <-writer.started:
	default:
		t.Fatal("expected writer to start")
	}
	close(writer.release)
}

func TestContextWithoutDeadlineGetsDefaultTimeout(t *testing.T) {
	ctx, cancel := contextWithDefaultTimeout(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected default deadline")
	}
	remaining := time.Until(deadline)
	if remaining < DefaultWriteTimeout-200*time.Millisecond || remaining > DefaultWriteTimeout+200*time.Millisecond {
		t.Fatalf("expected deadline near %s, got %s", DefaultWriteTimeout, remaining)
	}
}

func TestFormatLogEntryUsesLocalTimezoneAndMilliseconds(t *testing.T) {
	previousLocation := time.Local
	time.Local = time.FixedZone("test-local", 8*60*60)
	defer func() { time.Local = previousLocation }()

	instant := time.Date(2026, time.September, 19, 1, 2, 3, 456000000, time.UTC)
	entry := formatLogEntry(context.Background(), instant, LevelInfo, "example/worker/worker.go:42", "worker.Run", "POST /items -> 200", nil)
	want := "{\"timestamp\":\"2026-09-19 09:02:03.456\",\"level\":\"INFO\",\"file\":\"example/worker/worker.go:42\",\"method\":\"worker.Run\",\"message\":\"POST /items -> 200\"}\n"
	if entry != want {
		t.Fatalf("unexpected local timestamp format:\nwant %q\n got %q", want, entry)
	}
}

func TestContextLogIncludesTraceAndSpan(t *testing.T) {
	var output bytes.Buffer
	logger := newConsoleLogger(&output, io.Discard)
	ctx := WithTrace(context.Background(), "0123456789abcdef0123456789abcdef", "0123456789abcdef", "fedcba9876543210")

	if err := logger.InfoContext(ctx, "trace message", map[string]any{"operation": "test"}); err != nil {
		t.Fatalf("write trace log: %v", err)
	}

	var entry logEntry
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &entry); err != nil {
		t.Fatalf("decode trace log: %v", err)
	}
	if entry.TraceID != "0123456789abcdef0123456789abcdef" || entry.SpanID != "0123456789abcdef" || entry.ParentSpanID != "fedcba9876543210" {
		t.Fatalf("unexpected trace metadata: %+v", entry)
	}
	if entry.Attributes["operation"] != "test" {
		t.Fatalf("missing structured attributes: %+v", entry.Attributes)
	}
}

func TestStartSpanPreservesTraceAndSetsParent(t *testing.T) {
	parent := WithTrace(context.Background(), "0123456789abcdef0123456789abcdef", "0123456789abcdef", "")
	child := TraceFromContext(StartSpan(parent, "child-operation"))
	if child.TraceID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("trace ID changed: %+v", child)
	}
	if child.ParentSpanID != "0123456789abcdef" || !ValidSpanID(child.SpanID) || child.SpanID == child.ParentSpanID {
		t.Fatalf("invalid child span: %+v", child)
	}
	if child.SpanName != "child-operation" {
		t.Fatalf("missing child span name: %+v", child)
	}
}

func TestStandardLoggerUsesEnhancedFormat(t *testing.T) {
	var output bytes.Buffer
	logger := newConsoleLogger(io.Discard, &output)

	logger.StandardLogger(LevelError).Print("standard logger message")

	if !strings.Contains(output.String(), "standard logger message") {
		t.Fatalf("missing standard logger message: %s", output.String())
	}
	assertLogMetadata(t, output.String(), "logging_test.go", "logging.TestStandardLoggerUsesEnhancedFormat")
}

func assertLogMetadata(t *testing.T, value, file, method string) {
	t.Helper()
	timestampPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}$`)
	filePattern := regexp.MustCompile(`/` + regexp.QuoteMeta(file) + `:\d+$`)
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for _, line := range lines {
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode JSON log payload: %v; line=%q", err, line)
		}
		if !timestampPattern.MatchString(entry.Timestamp) {
			t.Fatalf("unexpected timestamp: %q", entry.Timestamp)
		}
		if entry.Level != LevelDebug && entry.Level != LevelInfo && entry.Level != LevelWarn && entry.Level != LevelError {
			t.Fatalf("unexpected level: %q", entry.Level)
		}
		if !strings.HasPrefix(entry.File, "web3-service-agent/internal/logging/") || !filePattern.MatchString(entry.File) || entry.Method != method {
			t.Fatalf("unexpected JSON log metadata: %+v", entry)
		}
	}
}

type blockingWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *blockingWriter) Write(payload []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return len(payload), nil
}

func assertFileContains(t *testing.T, path, marker string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file %s: %v", path, err)
	}
	if !strings.Contains(string(content), marker) {
		t.Fatalf("expected %q in %s, got %s", marker, path, string(content))
	}
}
