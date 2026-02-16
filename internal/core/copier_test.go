package core

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestCopier_Copy(t *testing.T) {
	// Setup in-memory filesystems
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	// Create a source file
	srcContent := []byte("Hello, CertiCopy!")
	srcPath := "/source/test.txt"
	if err := afero.WriteFile(srcFs, srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Initialize Copier
	copier := NewCopier(srcFs, dstFs)

	// Define destination path
	dstPath := "/dest/test.txt"
	// Create dest directory
	if err := dstFs.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Perform copy
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     false,
	}

	progressChan := make(chan Progress, 10)
	errChan := make(chan error)
	go func() {
		var err error
		defer func() {
			close(progressChan)
			errChan <- err
		}()
		err = copier.CopyWithProgress(srcPath, dstPath, opts, progressChan)
	}()

	// Consume progress
	var lastProgress Progress
	for p := range progressChan {
		lastProgress = p
	}

	if err := <-errChan; err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify content
	dstContent, err := afero.ReadFile(dstFs, dstPath)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}

	if string(dstContent) != string(srcContent) {
		t.Errorf("Content mismatch. Got %s, want %s", string(dstContent), string(srcContent))
	}

	// Verify checksum was calculated
	if lastProgress.SourceHash == "" {
		t.Error("Source hash was not calculated")
	}
}
