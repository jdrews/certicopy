package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestScanner_ScanSingleFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	srcFile := filepath.Join(base, "src", "file1.txt")
	dstDir := filepath.Join(base, "dest")
	afero.WriteFile(fs, srcFile, []byte("hello"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{srcFile}, dstDir)

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

	if files[0].SourcePath != srcFile {
		t.Errorf("Expected source %s, got %s", srcFile, files[0].SourcePath)
	}

	expectedDest := filepath.Join(dstDir, "file1.txt")
	if files[0].DestPath != expectedDest {
		t.Errorf("Expected dest %s, got %s", expectedDest, files[0].DestPath)
	}
}

func TestScanner_ScanDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "src", "dir")
	dstDir := filepath.Join(base, "dest")
	afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644)
	afero.WriteFile(fs, filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{srcDir}, dstDir)

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
			expectedDest := filepath.Join(dstDir, "dir", "file1.txt")
			if f.DestPath != expectedDest {
				t.Errorf("Expected dest %s for file1, got %s", expectedDest, f.DestPath)
			}
		}
		if f.Name == "file2.txt" {
			foundFile2 = true
			expectedDest := filepath.Join(dstDir, "dir", "subdir", "file2.txt")
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
	base, _ := filepath.Abs(".")
	srcFile1 := filepath.Join(base, "src", "file1.txt")
	srcFile2 := filepath.Join(base, "other", "file2.txt")
	dstDir := filepath.Join(base, "dest")
	afero.WriteFile(fs, srcFile1, []byte("a"), 0644)
	afero.WriteFile(fs, srcFile2, []byte("b"), 0644)

	s := NewScanner(fs)
	files, count, size, err := s.Scan([]string{srcFile1, srcFile2}, dstDir)

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
