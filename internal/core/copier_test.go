package core

import (
	"context"
	"path/filepath"
	"strings"
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
			errChan <- err
		}()
		ctx := context.Background()
		err = copier.CopyWithProgress(ctx, srcPath, dstPath, opts, progressChan)
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

func TestCopier_Copy_HashMismatch(t *testing.T) {
	// Setup in-memory filesystems
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	srcPath := "/source/test.txt"
	dstPath := "/dest/test.txt"

	// Create source file
	srcContent := []byte("Content for source file") // 23 bytes
	if err := srcFs.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := afero.WriteFile(srcFs, srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with SAME SIZE but DIFFERENT CONTENT
	dstContent := []byte("Content for destination") // 23 bytes
	if err := dstFs.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}
	if err := afero.WriteFile(dstFs, dstPath, dstContent, 0644); err != nil {
		t.Fatalf("Failed to create existing dest file: %v", err)
	}

	// Initialize Copier
	copier := NewCopier(srcFs, dstFs)

	// Perform copy with Overwrite: false and CalculateHash: true
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     false,
	}

	err := copier.Copy(srcPath, dstPath, opts)
	if err == nil {
		t.Fatal("Expected error due to hash mismatch, but got nil")
	}

	expectedErr := "[CHECKSUM_MISMATCH] destination exists with same size but different hash"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

// FaultyFs is a wrapper around afero.Fs that intercepts Create
type FaultyFs struct {
	afero.Fs
}

func (f *FaultyFs) Create(name string) (afero.File, error) {
	file, err := f.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return &FaultyFile{File: file}, nil
}

// FaultyFile is a wrapper around afero.File that corrupts data on Write
type FaultyFile struct {
	afero.File
}

func (f *FaultyFile) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		// Create a copy of the buffer to corrupt, so we don't mess up the T-Reader/Hasher
		corrupted := make([]byte, len(p))
		copy(corrupted, p)
		corrupted[0] = corrupted[0] + 1 // Corrupt the first byte
		return f.File.Write(corrupted)
	}
	return f.File.Write(p)
}

func TestCopier_Copy_PostCopyCorruption(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	// Use FaultyFs to simulate corruption on write
	dstFs := &FaultyFs{Fs: afero.NewMemMapFs()}

	srcPath := "/source/corrupt_me.txt"
	dstPath := "/dest/corrupt_me.txt"

	// Create source file
	srcContent := []byte("Strict verification needed")
	if err := srcFs.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := afero.WriteFile(srcFs, srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     true,
	}

	err := copier.Copy(srcPath, dstPath, opts)
	if err == nil {
		t.Fatal("Expected error due to post-copy checksum mismatch, but got nil")
	}

	if !strings.Contains(err.Error(), "[CHECKSUM_MISMATCH] checksum mismatch after copy") {
		t.Errorf("Expected error containing %q, got %q", "[CHECKSUM_MISMATCH] checksum mismatch after copy", err.Error())
	}
}
