package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"

	DefaultWriteTimeout = 2 * time.Second
	logTimestampLayout  = "2006-01-02 15:04:05.000"
)

type levelSink struct {
	mu     sync.Mutex
	writer io.Writer
}

func (s *levelSink) write(payload []byte) error {
	if s == nil || s.writer == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	written, err := s.writer.Write(payload)
	if err != nil {
		return err
	}
	if written != len(payload) {
		return io.ErrShortWrite
	}
	return nil
}

type Logger struct {
	debug   *levelSink
	info    *levelSink
	warn    *levelSink
	err     *levelSink
	closers []io.Closer
}

type logEntry struct {
	Timestamp    string         `json:"timestamp"`
	Level        Level          `json:"level"`
	TraceID      string         `json:"trace_id,omitempty"`
	SpanID       string         `json:"span_id,omitempty"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	SpanName     string         `json:"span_name,omitempty"`
	File         string         `json:"file"`
	Method       string         `json:"method"`
	Message      string         `json:"message"`
	Attributes   map[string]any `json:"attributes,omitempty"`
}

type contextLevelWriter struct {
	logger *Logger
	level  Level
}

func (w contextLevelWriter) Write(payload []byte) (int, error) {
	if w.logger == nil {
		return 0, errors.New("logger is nil")
	}
	if err := w.logger.writeContext(context.Background(), w.level, strings.TrimRight(string(payload), "\r\n"), nil); err != nil {
		return 0, err
	}
	return len(payload), nil
}

var (
	defaultMu     sync.RWMutex
	defaultLogger = newConsoleLogger(os.Stdout, os.Stderr)
)

func Setup(path string) (*Logger, error) {
	logger, err := New(path, os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	SetDefault(logger)
	return logger, nil
}

func New(path string, stdout, stderr io.Writer) (*Logger, error) {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	base := strings.TrimSpace(path)
	if base == "" {
		base = "."
	}
	logDir := filepath.Clean(base)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory %q: %w", logDir, err)
	}

	debugFile, err := openLogFile(logDir, "debug.log")
	if err != nil {
		return nil, err
	}
	infoFile, err := openLogFile(logDir, "info.log")
	if err != nil {
		_ = debugFile.Close()
		return nil, err
	}
	warnFile, err := openLogFile(logDir, "warn.log")
	if err != nil {
		_ = debugFile.Close()
		_ = infoFile.Close()
		return nil, err
	}
	errorFile, err := openLogFile(logDir, "error.log")
	if err != nil {
		_ = debugFile.Close()
		_ = infoFile.Close()
		_ = warnFile.Close()
		return nil, err
	}

	return &Logger{
		debug:   newLevelSink(io.MultiWriter(stdout, debugFile)),
		info:    newLevelSink(io.MultiWriter(stdout, infoFile)),
		warn:    newLevelSink(io.MultiWriter(stderr, warnFile)),
		err:     newLevelSink(io.MultiWriter(stderr, errorFile)),
		closers: []io.Closer{debugFile, infoFile, warnFile, errorFile},
	}, nil
}

func newConsoleLogger(stdout, stderr io.Writer) *Logger {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &Logger{
		debug: newLevelSink(stdout),
		info:  newLevelSink(stdout),
		warn:  newLevelSink(stderr),
		err:   newLevelSink(stderr),
	}
}

func newLevelSink(writer io.Writer) *levelSink {
	return &levelSink{writer: writer}
}

func openLogFile(dir, name string) (*os.File, error) {
	filePath := filepath.Join(dir, name)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", filePath, err)
	}
	return file, nil
}

func SetDefault(logger *Logger) {
	if logger == nil {
		return
	}
	defaultMu.Lock()
	defaultLogger = logger
	defaultMu.Unlock()
}

func Default() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	var errs []string
	for _, closer := range l.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close log files: %s", strings.Join(errs, "; "))
	}
	return nil
}

// StandardLogger returns a standard-library logger whose writes still receive
// the timestamp, source location, method name, level, and default timeout.
func (l *Logger) StandardLogger(level Level) *log.Logger {
	return log.New(contextLevelWriter{logger: l, level: normalizeLevel(level)}, "", 0)
}

func StandardLogger(level Level) *log.Logger {
	return Default().StandardLogger(level)
}

// Context variants respect an existing context deadline. When ctx has no
// deadline (or is nil), the write is automatically limited to two seconds.
func (l *Logger) DebugfContext(ctx context.Context, format string, args ...any) error {
	return l.writeContext(ctx, LevelDebug, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) InfofContext(ctx context.Context, format string, args ...any) error {
	return l.writeContext(ctx, LevelInfo, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) WarnfContext(ctx context.Context, format string, args ...any) error {
	return l.writeContext(ctx, LevelWarn, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) ErrorfContext(ctx context.Context, format string, args ...any) error {
	return l.writeContext(ctx, LevelError, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) DebugContext(ctx context.Context, message string, attributes map[string]any) error {
	return l.writeContext(ctx, LevelDebug, message, attributes)
}

func (l *Logger) InfoContext(ctx context.Context, message string, attributes map[string]any) error {
	return l.writeContext(ctx, LevelInfo, message, attributes)
}

func (l *Logger) WarnContext(ctx context.Context, message string, attributes map[string]any) error {
	return l.writeContext(ctx, LevelWarn, message, attributes)
}

func (l *Logger) ErrorContext(ctx context.Context, message string, attributes map[string]any) error {
	return l.writeContext(ctx, LevelError, message, attributes)
}

func (l *Logger) Debugf(format string, args ...any) {
	_ = l.DebugfContext(context.Background(), format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	_ = l.InfofContext(context.Background(), format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	_ = l.WarnfContext(context.Background(), format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	_ = l.ErrorfContext(context.Background(), format, args...)
}

func DebugfContext(ctx context.Context, format string, args ...any) error {
	return Default().DebugfContext(ctx, format, args...)
}

func InfofContext(ctx context.Context, format string, args ...any) error {
	return Default().InfofContext(ctx, format, args...)
}

func WarnfContext(ctx context.Context, format string, args ...any) error {
	return Default().WarnfContext(ctx, format, args...)
}

func ErrorfContext(ctx context.Context, format string, args ...any) error {
	return Default().ErrorfContext(ctx, format, args...)
}

func DebugContext(ctx context.Context, message string, attributes map[string]any) error {
	return Default().DebugContext(ctx, message, attributes)
}

func InfoContext(ctx context.Context, message string, attributes map[string]any) error {
	return Default().InfoContext(ctx, message, attributes)
}

func WarnContext(ctx context.Context, message string, attributes map[string]any) error {
	return Default().WarnContext(ctx, message, attributes)
}

func ErrorContext(ctx context.Context, message string, attributes map[string]any) error {
	return Default().ErrorContext(ctx, message, attributes)
}

func Debugf(format string, args ...any) { Default().Debugf(format, args...) }
func Infof(format string, args ...any)  { Default().Infof(format, args...) }
func Warnf(format string, args ...any)  { Default().Warnf(format, args...) }
func Errorf(format string, args ...any) { Default().Errorf(format, args...) }

func (l *Logger) writeContext(ctx context.Context, level Level, message string, attributes map[string]any) error {
	if l == nil {
		return errors.New("logger is nil")
	}
	ctx, cancel := contextWithDefaultTimeout(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	file, method := callerMetadata()
	payload := formatLogEntry(ctx, time.Now(), normalizeLevel(level), file, method, message, attributes)
	result := make(chan error, 1)
	sink := l.sink(level)
	go func() {
		result <- sink.write([]byte(payload))
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func contextWithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultWriteTimeout)
}

func (l *Logger) sink(level Level) *levelSink {
	switch normalizeLevel(level) {
	case LevelDebug:
		return l.debug
	case LevelWarn:
		return l.warn
	case LevelError:
		return l.err
	default:
		return l.info
	}
}

func normalizeLevel(level Level) Level {
	switch level {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return level
	default:
		return LevelInfo
	}
}

func formatLogEntry(ctx context.Context, now time.Time, level Level, file, method, message string, attributes map[string]any) string {
	timestamp := now.In(time.Local).Format(logTimestampLayout)
	trace := TraceFromContext(ctx)
	message = strings.TrimRight(message, "\r\n")
	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	var output strings.Builder
	for _, entry := range lines {
		entry = strings.TrimSuffix(entry, "\r")
		payload := encodeLogEntry(logEntry{
			Timestamp:    timestamp,
			Level:        level,
			TraceID:      trace.TraceID,
			SpanID:       trace.SpanID,
			ParentSpanID: trace.ParentSpanID,
			SpanName:     trace.SpanName,
			File:         file,
			Method:       method,
			Message:      entry,
			Attributes:   attributes,
		})
		output.WriteString(payload)
	}
	return output.String()
}

func encodeLogEntry(entry logEntry) string {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(entry); err != nil {
		return "{\"timestamp\":\"\",\"level\":\"ERROR\",\"file\":\"internal/logging\",\"method\":\"encodeLogEntry\",\"message\":\"encode log entry failed\"}\n"
	}
	return output.String()
}

func callerMetadata() (string, string) {
	programCounters := make([]uintptr, 32)
	count := runtime.Callers(2, programCounters)
	frames := runtime.CallersFrames(programCounters[:count])
	for {
		frame, more := frames.Next()
		if !internalLoggingFrame(frame) {
			return sourceLocation(frame), shortFunctionName(frame.Function)
		}
		if !more {
			break
		}
	}
	return "unknown:0", "unknown"
}

func sourceLocation(frame runtime.Frame) string {
	file := fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
	packageName := packagePath(frame.Function)
	if packageName == "" {
		return file
	}
	return strings.TrimSuffix(packageName, "/") + "/" + file
}

func packagePath(function string) string {
	slash := strings.LastIndex(function, "/")
	packageAndMethod := function[slash+1:]
	dot := strings.Index(packageAndMethod, ".")
	if dot < 0 {
		return ""
	}
	return function[:slash+1+dot]
}

func internalLoggingFrame(frame runtime.Frame) bool {
	file := filepath.ToSlash(frame.File)
	if strings.HasSuffix(file, "/internal/logging/logging.go") {
		return true
	}
	if strings.HasSuffix(file, "/src/log/log.go") {
		return true
	}
	return strings.HasPrefix(frame.Function, "runtime.")
}

func shortFunctionName(function string) string {
	if slash := strings.LastIndex(function, "/"); slash >= 0 {
		function = function[slash+1:]
	}
	return function
}
