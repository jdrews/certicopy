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

// Scan walks the source paths and returns a list of files to transfer, total size, and total count
func (s *Scanner) Scan(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
	var files []*models.FileInfo
	var totalSize int64
	var totalCount int64

	for _, source := range sources {
		// Get absolute path for source
		// Note: We assume source is strictly within the fs provided to Scanner
		// If fs is specific to a directory, paths must be relative?
		// Usually we assume fs is OsFs for the whole system in desktop apps.

		info, err := s.fs.Stat(source)
		if err != nil {
			return nil, 0, 0, err
		}

		if !info.IsDir() {
			// Single file
			destPath := filepath.Join(destRoot, info.Name())
			file := &models.FileInfo{
				SourcePath: source,
				DestPath:   destPath,
				Name:       info.Name(),
				Size:       info.Size(),
				ModTime:    info.ModTime().UnixMilli(),
				Status:     models.StatusPending,
			}
			files = append(files, file)
			totalSize += info.Size()
			totalCount++
			continue
		}

		// Directory - walk it
		// We want to preserve the directory structure relative to the source parent?
		// e.g. cp -r /a/b /c -> /c/b/...
		// Other tools like this usually copy the folder itself if selected.
		// TODO: preserver director structure. remove references to TeraCopy
		sourceName := filepath.Base(source)

		err = afero.Walk(s.fs, source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			// Calculate relative path from source
			relPath, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}

			// Destination path includes the source folder name
			destPath := filepath.Join(destRoot, sourceName, relPath)

			file := &models.FileInfo{
				SourcePath: path,
				DestPath:   destPath,
				Name:       info.Name(),
				Size:       info.Size(),
				ModTime:    info.ModTime().UnixMilli(),
				Status:     models.StatusPending,
			}
			files = append(files, file)
			totalSize += info.Size()
			totalCount++
			return nil
		})
		if err != nil {
			return nil, 0, 0, err
		}
	}

	return files, totalCount, totalSize, nil
}
