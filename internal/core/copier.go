package core

import (
	"fmt"
	"hash"
	"io"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

// CopyOptions defines configuration for the copy operation
type CopyOptions struct {
	BufferSize    int // Size in bytes, default 1MB (1024*1024)
	PreservePerms bool
	PreserveTimes bool
	CalculateHash bool
	HashAlgorithm HashAlgorithm
	Overwrite     bool
}

// Progress update struct
type Progress struct {
	BytesCopied int64   `json:"bytesCopied"`
	TotalBytes  int64   `json:"totalBytes"`
	Speed       float64 `json:"speed"` // bytes per second
	SourceHash  string  `json:"sourceHash"`
	DestHash    string  `json:"destHash"`
}

// Copier handles file copy operations
type Copier struct {
	srcFs afero.Fs
	dstFs afero.Fs
}

// NewCopier creates a new Copier with source and destination filesystems
func NewCopier(srcFs, dstFs afero.Fs) *Copier {
	return &Copier{
		srcFs: srcFs,
		dstFs: dstFs,
	}
}

// Copy performs a file copy with the given options
func (c *Copier) Copy(src string, dst string, opts CopyOptions) error {
	progressChan := make(chan Progress)
	// Drain the channel to prevent blocking if caller doesn't read
	go func() {
		for range progressChan {
		}
	}()
	return c.CopyWithProgress(src, dst, opts, progressChan)
}

// CopyWithProgress performs a file copy and streams progress updates
func (c *Copier) CopyWithProgress(src string, dst string, opts CopyOptions, progressChan chan<- Progress) error {
	defer close(progressChan)
	fmt.Printf("Copier: Start copying %s to %s\n", src, dst)

	// ## Open source file
	srcFile, err := c.srcFs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// ## Get source info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// ## Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := c.dstFs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// ## Check destination
	if exists, _ := afero.Exists(c.dstFs, dst); exists {
		if !opts.Overwrite {
			return fmt.Errorf("destination exists and overwrite is false")
		}
		// Warning: If we don't truncate or remove, io.Copy might behave differently depending on flags.
		// afero.Create forces truncation.
	}

	// ## Create destination file
	dstFile, err := c.dstFs.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// ## Allow specifying buffer size
	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 1024 * 1024 // Default 1MB
	}
	buf := make([]byte, bufSize)

	// ## Setup streaming checksum
	var hasher hash.Hash
	var hashResult string
	if opts.CalculateHash {
		h, err := NewHasher(opts.HashAlgorithm)
		if err != nil {
			return err
		}
		hasher = h
	}

	// ## Setup progress tracking
	startTime := time.Now()
	totalBytes := srcInfo.Size()

	// Create a proxy reader that updates progress and writes to hasher
	proxyReader := &ReaderProxy{
		Reader: srcFile,
		Total:  totalBytes,
		Context: &ProgressContext{
			Channel:   progressChan,
			StartTime: startTime,
		},
	}

	// If calculating hash, TeeReader so data goes to both Hasher and Destination
	var reader io.Reader = proxyReader
	if opts.CalculateHash && hasher != nil {
		reader = io.TeeReader(proxyReader, hasher)
	}

	// ## Perform Copy
	// Use io.CopyBuffer for buffered copy
	written, err := io.CopyBuffer(dstFile, reader, buf)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// Calculate source hash immediately after copy
	if hasher != nil {
		hashResult = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	// ## Calculate destination hash for verification
	var destHashResult string
	if opts.CalculateHash {
		dstHasher, err := NewHasher(opts.HashAlgorithm)
		if err == nil {
			// Re-open destination file to hash it
			dstReadFile, err := c.dstFs.Open(dst)
			if err == nil {
				defer dstReadFile.Close()
				if _, err := io.Copy(dstHasher, dstReadFile); err == nil {
					destHashResult = fmt.Sprintf("%x", dstHasher.Sum(nil))
				}
			}
		}
	}

	// ## Send final progress
	if progressChan != nil {
		progressChan <- Progress{
			BytesCopied: written,
			TotalBytes:  totalBytes,
			Speed:       float64(written) / time.Since(startTime).Seconds(),
			SourceHash:  hashResult,
			DestHash:    destHashResult,
		}
	}

	// ## Preserve Metadata
	if opts.PreservePerms {
		if err := c.dstFs.Chmod(dst, srcInfo.Mode()); err != nil {
			// Log warning?
		}
	}
	if opts.PreserveTimes {
		if err := c.dstFs.Chtimes(dst, time.Now(), srcInfo.ModTime()); err != nil {
			// Log warning?
		}
	}

	return nil
}

type ProgressContext struct {
	Channel   chan<- Progress
	StartTime time.Time
	BytesRead int64
}

type ReaderProxy struct {
	Reader  io.Reader
	Total   int64
	Context *ProgressContext
}

func (r *ReaderProxy) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Context.BytesRead += int64(n)

	// Emit progress
	// Note: In production, we might want to throttle this to every 500ms
	// For now, we'll let the receiver handle throttling or add a ticker here
	select {
	case r.Context.Channel <- Progress{
		BytesCopied: r.Context.BytesRead,
		TotalBytes:  r.Total,
		Speed:       float64(r.Context.BytesRead) / time.Since(r.Context.StartTime).Seconds(),
	}:
	default:
		// Non-blocking send to avoid slowing down copy if channel is full
	}

	return
}
