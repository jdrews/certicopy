package core

import (
	"context"
	"testing"

	"github.com/spf13/afero"
)

func TestCopier_OverwritePriority(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	srcContent := []byte("Original Content")
	srcPath := "/source/test.txt"
	afero.WriteFile(srcFs, srcPath, srcContent, 0644)

	dstPath := "/dest/test.txt"
	// Create destination with SAME SIZE but DIFFERENT content
	afero.WriteFile(dstFs, dstPath, []byte("DifferentContent"), 0644)

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     true, // Should overwrite even if sizes match
	}

	err := copier.CopyWithProgress(context.Background(), srcPath, dstPath, opts, nil)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	content, _ := afero.ReadFile(dstFs, dstPath)
	if string(content) != string(srcContent) {
		t.Errorf("Overwrite failed. Got %s, want %s", string(content), string(srcContent))
	}
}

func TestCopier_HashValidation(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	content := []byte("Same Content")
	srcPath := "/source/test.txt"
	afero.WriteFile(srcFs, srcPath, content, 0644)

	dstPath := "/dest/test.txt"
	afero.WriteFile(dstFs, dstPath, content, 0644)

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     false,
	}

	progressChan := make(chan Progress, 1)
	err := copier.CopyWithProgress(context.Background(), srcPath, dstPath, opts, progressChan)
	if err != nil {
		t.Fatalf("Copy should have skipped but failed: %v", err)
	}

	// In this case, it should return nil and indicate success
}

func TestCopier_HashMismatchError(t *testing.T) {
	srcFs := afero.NewMemMapFs()
	dstFs := afero.NewMemMapFs()

	srcPath := "/source/test.txt"
	afero.WriteFile(srcFs, srcPath, []byte("Content A"), 0644)

	dstPath := "/dest/test.txt"
	// Same size but different content
	afero.WriteFile(dstFs, dstPath, []byte("Content B"), 0644)

	copier := NewCopier(srcFs, dstFs)
	opts := CopyOptions{
		BufferSize:    1024,
		CalculateHash: true,
		HashAlgorithm: HashSHA256,
		Overwrite:     false, // Should error because hashes mismatch
	}

	err := copier.CopyWithProgress(context.Background(), srcPath, dstPath, opts, nil)
	if err == nil {
		t.Fatal("Expected error due to hash mismatch, but got nil")
	}
}
