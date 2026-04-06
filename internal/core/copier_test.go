package core

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func TestCopier_HashWithProgress(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	content := []byte("constant content")
	srcPath := "/s/f.txt"
	dstPath := "/d/f.txt"

	afero.WriteFile(srcFs, srcPath, content, 0644)
	afero.WriteFile(dstFs, dstPath, content, 0644)

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{BufferSize: 1024, HashAlgorithm: HashMD5}
	progressChan := make(chan Progress, 100)

	srcHash, dstHash, err := copier.HashWithProgress(context.Background(), srcPath, dstPath, opts, progressChan)
	if err != nil {
		t.Fatalf("HashWithProgress failed: %v", err)
	}

	expectedHash := "b89cc4306e89c18d185fa217eb6b2120"
	if srcHash != expectedHash || dstHash != expectedHash {
		t.Errorf("Hash mismatch. Got %s, %s; want %s", srcHash, dstHash, expectedHash)
	}

	// Verify progress was sent
	count := 0
	for range progressChan {
		count++
	}
	if count == 0 {
		t.Error("No progress updates received")
	}
}

func TestCopier_Resume(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	srcContent := []byte("full content of the file") // 24 bytes
	srcPath := "/src/file.bin"
	dstPath := "/dst/file.bin"

	afero.WriteFile(srcFs, srcPath, srcContent, 0644)
	// Partial content already at destination
	afero.WriteFile(dstFs, dstPath, srcContent[:10], 0644)

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{
		BufferSize:    1024,
		Resume:        true,
		CalculateHash: true,
		HashAlgorithm: HashMD5,
	}

	progressChan := make(chan Progress, 100)
	err := copier.CopyWithProgress(context.Background(), srcPath, dstPath, opts, progressChan)
	if err != nil {
		t.Fatalf("Resume copy failed: %v", err)
	}

	// Verify final content
	dstContent, _ := afero.ReadFile(dstFs, dstPath)
	if string(dstContent) != string(srcContent) {
		t.Errorf("Resume failed to complete file. Got %q, want %q", string(dstContent), string(srcContent))
	}
}

type errorReader struct {
	io.Reader
	remaining int
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.remaining <= 0 {
		return 0, errors.New("read error")
	}
	n, err = r.Reader.Read(p[:min(len(p), r.remaining)])
	r.remaining -= n
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type errorReaderFs struct {
	afero.Fs
	errorAt int
}

func (f *errorReaderFs) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	return &errorReaderFile{File: file, errorAt: f.errorAt}, nil
}

type errorReaderFile struct {
	afero.File
	errorAt int
	readSoFar int
}

func (f *errorReaderFile) Read(p []byte) (n int, err error) {
	if f.readSoFar >= f.errorAt {
		return 0, fmt.Errorf("simulated read error at %d", f.readSoFar)
	}
	n, err = f.File.Read(p)
	f.readSoFar += n
	return
}

func TestCopier_Copy_ReadError(t *testing.T) {
	srcFs := &errorReaderFs{Fs: afero.NewMemMapFs(), errorAt: 10}
	dstFs := afero.NewMemMapFs()

	srcPath := "/s/f.txt"
	afero.WriteFile(srcFs.Fs, srcPath, []byte("large file content that will fail eventually"), 0644)

	copier := NewCopier(srcFs, dstFs)
	err := copier.Copy(srcPath, "/d/f.txt", CopyOptions{BufferSize: 5})
	if err == nil {
		t.Fatal("Expected read error, got nil")
	}
	if !strings.Contains(err.Error(), "simulated read error") {
		t.Errorf("Expected simulated read error, got %v", err)
	}
}

