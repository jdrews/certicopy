package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLogger(t *testing.T) {
	// Test terminal-only logging
	err := InitLogger("")
	if err != nil {
		t.Fatalf("InitLogger failed (terminal only): %v", err)
	}

	// Test file logging
	tempDir, err := os.MkdirTemp("", "certicopy-log-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logPath := filepath.Join(tempDir, "test.log")
	err = InitLogger(logPath)
	if err != nil {
		t.Fatalf("InitLogger failed (file mode): %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestInitLogger_Error(t *testing.T) {
	// Test invalid path (non-existent directory)
	err := InitLogger("/non/existent/path/to/log.log")
	if err == nil {
		t.Error("Expected error for invalid log path, got nil")
	}
}
