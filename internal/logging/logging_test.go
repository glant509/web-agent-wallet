package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	if !strings.Contains(stdout.String(), "DEBUG debug message") {
		t.Fatalf("stdout missing debug message: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "INFO info message") {
		t.Fatalf("stdout missing info message: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "WARN warn message") {
		t.Fatalf("stderr missing warn message: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ERROR error message") {
		t.Fatalf("stderr missing error message: %s", stderr.String())
	}
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
