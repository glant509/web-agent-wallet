package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Logger struct {
	debug   *log.Logger
	info    *log.Logger
	warn    *log.Logger
	err     *log.Logger
	closers []io.Closer
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
	logDir := filepath.Join(base, "logs")
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
		debug:   newLevelLogger(LevelDebug, io.MultiWriter(stdout, debugFile)),
		info:    newLevelLogger(LevelInfo, io.MultiWriter(stdout, infoFile)),
		warn:    newLevelLogger(LevelWarn, io.MultiWriter(stderr, warnFile)),
		err:     newLevelLogger(LevelError, io.MultiWriter(stderr, errorFile)),
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
		debug: newLevelLogger(LevelDebug, stdout),
		info:  newLevelLogger(LevelInfo, stdout),
		warn:  newLevelLogger(LevelWarn, stderr),
		err:   newLevelLogger(LevelError, stderr),
	}
}

func openLogFile(dir, name string) (*os.File, error) {
	filePath := filepath.Join(dir, name)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", filePath, err)
	}
	return file, nil
}

func newLevelLogger(level Level, writer io.Writer) *log.Logger {
	return log.New(writer, string(level)+" ", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
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

func (l *Logger) StandardLogger(level Level) *log.Logger {
	switch level {
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

func StandardLogger(level Level) *log.Logger {
	return Default().StandardLogger(level)
}

func (l *Logger) Debugf(format string, args ...any) { l.debug.Printf(format, args...) }
func (l *Logger) Infof(format string, args ...any)  { l.info.Printf(format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.warn.Printf(format, args...) }
func (l *Logger) Errorf(format string, args ...any) { l.err.Printf(format, args...) }

func Debugf(format string, args ...any) { Default().Debugf(format, args...) }
func Infof(format string, args ...any)  { Default().Infof(format, args...) }
func Warnf(format string, args ...any)  { Default().Warnf(format, args...) }
func Errorf(format string, args ...any) { Default().Errorf(format, args...) }
