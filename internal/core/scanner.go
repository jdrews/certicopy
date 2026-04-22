package core

import (
	"os"
	"path/filepath"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/spf13/afero"
)

// Scanner handles directory scanning and file enumeration
type Scanner struct {
	fs afero.Fs
}

// NewScanner creates a new Scanner
func NewScanner(fs afero.Fs) *Scanner {
	return &Scanner{fs: fs}
}

// ScanResult holds the summarized findings of a scan
type ScanResult struct {
	Files []*models.FileInfo
	Count int64
	Size  int64
}

func (s *Scanner) Scan(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
	result := &ScanResult{}

	// Normalize destination root to absolute path for Windows long path support
	absDestRoot, err := filepath.Abs(destRoot)
	if err == nil {
		destRoot = absDestRoot
	}

	for _, source := range sources {
		// Normalize source to absolute path
		absSource, err := filepath.Abs(source)
		if err == nil {
			source = absSource
		}

		info, err := s.fs.Stat(source)
		if err != nil {
			return nil, 0, 0, err
		}

		if !info.IsDir() {
			s.handleSingleFile(source, destRoot, info, result)
			continue
		}

		if err := s.handleDirectory(source, destRoot, result); err != nil {
			return nil, 0, 0, err
		}
	}

	return result.Files, result.Count, result.Size, nil
}

func (s *Scanner) handleSingleFile(source string, destRoot string, info os.FileInfo, result *ScanResult) {
	destPath := filepath.Join(destRoot, info.Name())
	file := &models.FileInfo{
		SourcePath: source,
		DestPath:   destPath,
		Name:       info.Name(),
		Size:       info.Size(),
		ModTime:    info.ModTime().UnixMilli(),
		Status:     models.StatusPending,
	}
	result.Files = append(result.Files, file)
	result.Size += info.Size()
	result.Count++
}

func (s *Scanner) handleDirectory(source string, destRoot string, result *ScanResult) error {
	sourceName := filepath.Base(source)

	return afero.Walk(s.fs, source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destRoot, sourceName, relPath)
		file := &models.FileInfo{
			SourcePath: path,
			DestPath:   destPath,
			Name:       info.Name(),
			Size:       info.Size(),
			ModTime:    info.ModTime().UnixMilli(),
			Status:     models.StatusPending,
		}
		result.Files = append(result.Files, file)
		result.Size += info.Size()
		result.Count++
		return nil
	})
}
