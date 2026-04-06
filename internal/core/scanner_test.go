package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestScanner_ScanSingleFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/file1.txt", []byte("hello"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{"/src/file1.txt"}, "/dest")

	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 file, got %d", count)
	}

	if size != 5 {
		t.Errorf("Expected 5 bytes, got %d", size)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file in result, got %d", len(files))
	}

	if files[0].SourcePath != "/src/file1.txt" {
		t.Errorf("Expected source /src/file1.txt, got %s", files[0].SourcePath)
	}

	if files[0].DestPath != filepath.FromSlash("/dest/file1.txt") {
		t.Errorf("Expected dest /dest/file1.txt, got %s", files[0].DestPath)
	}
}

func TestScanner_ScanDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/dir/file1.txt", []byte("file1"), 0644)
	afero.WriteFile(fs, "/src/dir/subdir/file2.txt", []byte("file2"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{"/src/dir"}, "/dest")

	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files, got %d", count)
	}

	if size != 10 {
		t.Errorf("Expected 10 bytes, got %d", size)
	}

	// Verify paths
	// Expected dest paths: /dest/dir/file1.txt, /dest/dir/subdir/file2.txt
	foundFile1 := false
	foundFile2 := false
	for _, f := range files {
		if f.Name == "file1.txt" {
			foundFile1 = true
			expectedDest := filepath.FromSlash("/dest/dir/file1.txt")
			if f.DestPath != expectedDest {
				t.Errorf("Expected dest %s for file1, got %s", expectedDest, f.DestPath)
			}
		}
		if f.Name == "file2.txt" {
			foundFile2 = true
			expectedDest := filepath.FromSlash("/dest/dir/subdir/file2.txt")
			if f.DestPath != expectedDest {
				t.Errorf("Expected dest %s for file2, got %s", expectedDest, f.DestPath)
			}
		}
	}

	if !foundFile1 || !foundFile2 {
		t.Error("Did not find all files")
	}
}

func TestScanner_ScanMultipleSources(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/file1.txt", []byte("a"), 0644)
	afero.WriteFile(fs, "/other/file2.txt", []byte("b"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{"/src/file1.txt", "/other/file2.txt"}, "/dest")

	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files, got %d", count)
	}

	if size != 2 {
		t.Errorf("Expected 2 bytes, got %d", size)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files in result, got %d", len(files))
	}
}

func TestScanner_ScanError(t *testing.T) {
	fs := afero.NewMemMapFs()
	s := NewScanner(fs)

	_, _, _, err := s.Scan([]string{"/nonexistent"}, "/dest")
	if err == nil {
		t.Error("Expected error for non-existent source, got nil")
	}
}

type errorFs struct {
	afero.Fs
}

func (e *errorFs) Stat(name string) (os.FileInfo, error) {
	return nil, os.ErrPermission
}

func TestScanner_ScanStatError(t *testing.T) {
	fs := &errorFs{Fs: afero.NewMemMapFs()}
	s := NewScanner(fs)

	_, _, _, err := s.Scan([]string{"/somefile"}, "/dest")
	if err != os.ErrPermission {
		t.Errorf("Expected permission error, got %v", err)
	}
}
