package core

import (
	"context"
	"fmt"
	"hash"
	"io"
	"os"
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
	Resume        bool
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
	return c.CopyWithProgress(context.Background(), src, dst, opts, progressChan)
}

// CopyWithProgress performs a file copy and streams progress updates.
// Supports context cancellation and byte-level resume.
func (c *Copier) CopyWithProgress(ctx context.Context, src string, dst string, opts CopyOptions, progressChan chan<- Progress) error {
	if progressChan != nil {
		defer close(progressChan)
	}
	fmt.Printf("Copier: Start copying %s to %s (Resume: %v)\n", src, dst, opts.Resume)

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
	totalBytes := srcInfo.Size()

	// ## Handle Overwrite / Resume / Existing File
	var startOffset int64
	var isAppending bool

	if !opts.Overwrite {
		if exists, _ := afero.Exists(c.dstFs, dst); exists {
			dstInfo, err := c.dstFs.Stat(dst)
			if err == nil {
				if opts.Resume && dstInfo.Size() < totalBytes {
					startOffset = dstInfo.Size()
					isAppending = true
					fmt.Printf("Copier: Resuming at offset %d\n", startOffset)
				} else if dstInfo.Size() == totalBytes {
					// Leaning on hash results to know if a file has been fully transferred
					fmt.Println("Copier: File sizes match. Calculating hashes for verification...")
					srcHash, err := CalculateChecksum(c.srcFs, src, opts.HashAlgorithm)
					if err != nil {
						return fmt.Errorf("failed to calculate source hash: %w", err)
					}
					dstHash, err := CalculateChecksum(c.dstFs, dst, opts.HashAlgorithm)
					if err != nil {
						return fmt.Errorf("failed to calculate destination hash: %w", err)
					}

					if srcHash == dstHash {
						fmt.Println("Copier: Hashes match. File already fully copied.")
						// Send completion progress
						if progressChan != nil {
							progressChan <- Progress{
								BytesCopied: totalBytes,
								TotalBytes:  totalBytes,
								SourceHash:  srcHash,
								DestHash:    dstHash,
							}
						}
						return nil
					}
					return fmt.Errorf("destination exists with same size but different hash")
				} else if dstInfo.Size() > totalBytes {
					return fmt.Errorf("destination exists and is larger than source")
				} else if !opts.Resume {
					return fmt.Errorf("destination exists and overwrite is false")
				}
			}
		}
	}

	// ## Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := c.dstFs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// ## Open/Create destination file
	var dstFile afero.File
	if isAppending {
		// Open for appending/resume
		dstFile, err = c.dstFs.OpenFile(dst, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open destination for resume: %w", err)
		}

		// Seek source file to startOffset
		seeker, ok := srcFile.(io.Seeker)
		if !ok {
			return fmt.Errorf("source file does not support seeking")
		}
		if _, err := seeker.Seek(startOffset, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek source file: %w", err)
		}
	} else {
		// New file or overwrite
		// If resume is true and file exists, it was likely handled above.
		// If it reaches here and exists, it means either:
		// 1. Resume is false and file exists (must check Overwrite)
		// 2. Resume is true but file size check failed (must check Overwrite)
		if exists, _ := afero.Exists(c.dstFs, dst); exists && !opts.Overwrite {
			return fmt.Errorf("destination exists and overwrite is false")
		}

		dstFile, err = c.dstFs.Create(dst)
		if err != nil {
			return fmt.Errorf("failed to create destination: %w", err)
		}
	}
	defer dstFile.Close()

	// ## Setup streaming checksum (Note: only hashes NEW data during copy)
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

	// Create a proxy reader that updates progress, writes to hasher, and checks context
	proxyReader := &ReaderProxy{
		Reader:  srcFile,
		Total:   totalBytes,
		Context: ctx,
		ProgressCtx: &ProgressContext{
			Channel:     progressChan,
			StartTime:   startTime,
			BytesRead:   startOffset,
			TotalOffset: startOffset,
		},
	}

	// If calculating hash, TeeReader so data goes to both Hasher and Destination
	var reader io.Reader = proxyReader
	if opts.CalculateHash && hasher != nil {
		reader = io.TeeReader(proxyReader, hasher)
	}

	// ## Perform Copy
	buf := make([]byte, opts.BufferSize)
	if len(buf) == 0 {
		buf = make([]byte, 1024*1024) // 1MB default
	}

	written, err := io.CopyBuffer(dstFile, reader, buf)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// ## Verification Logic: If we resumed, we MUST re-hash the source file fully
	if opts.CalculateHash {
		if startOffset > 0 {
			fmt.Println("Copier: Performing mandatory full re-hash for resume")
			fullSourceHash, err := CalculateChecksum(c.srcFs, src, opts.HashAlgorithm)
			if err != nil {
				return fmt.Errorf("failed to calculate full source hash: %w", err)
			}
			hashResult = fullSourceHash
		} else if hasher != nil {
			hashResult = fmt.Sprintf("%x", hasher.Sum(nil))
		}
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
			BytesCopied: startOffset + written,
			TotalBytes:  totalBytes,
			Speed:       float64(written) / time.Since(startTime).Seconds(),
			SourceHash:  hashResult,
			DestHash:    destHashResult,
		}
	}

	// ## Post-copy verification
	if opts.CalculateHash && hashResult != "" && destHashResult != "" {
		if hashResult != destHashResult {
			return fmt.Errorf("checksum mismatch after copy: source %s, dest %s", hashResult, destHashResult)
		}
	}

	// ## Preserve Metadata
	if opts.PreservePerms {
		if err := c.dstFs.Chmod(dst, srcInfo.Mode()); err != nil {
			// Log warning
		}
	}
	if opts.PreserveTimes {
		// NOTE: Be careful with Chtimes for partial copies.
		// For now, only preserve times on successful full completion.
		if err := c.dstFs.Chtimes(dst, time.Now(), srcInfo.ModTime()); err != nil {
			// Log warning
		}
	}

	return nil
}

type ProgressContext struct {
	Channel     chan<- Progress
	StartTime   time.Time
	BytesRead   int64
	TotalOffset int64
}

type ReaderProxy struct {
	Reader      io.Reader
	Total       int64
	Context     context.Context
	ProgressCtx *ProgressContext
}

func (r *ReaderProxy) Read(p []byte) (n int, err error) {
	// Check for cancellation before each read
	select {
	case <-r.Context.Done():
		return 0, r.Context.Err()
	default:
	}

	n, err = r.Reader.Read(p)
	r.ProgressCtx.BytesRead += int64(n)

	// Emit progress
	// Note: In production, we might want to throttle this to every 500ms
	// For now, we'll let the receiver handle throttling or add a ticker here
	select {
	case r.ProgressCtx.Channel <- Progress{
		BytesCopied: r.ProgressCtx.BytesRead,
		TotalBytes:  r.Total,
		Speed:       float64(r.ProgressCtx.BytesRead-r.ProgressCtx.TotalOffset) / time.Since(r.ProgressCtx.StartTime).Seconds(),
	}:
	default:
	}

	return
}
