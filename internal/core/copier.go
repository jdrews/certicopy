package core

import (
	"context"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/sirupsen/logrus"
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
	Log.WithFields(logrus.Fields{
		"src":    src,
		"dst":    dst,
		"resume": opts.Resume,
	}).Info("Starting file copy")

	// ## Open source file
	srcFile, err := c.srcFs.Open(src)
	if err != nil {
		return MapError(err, src, "failed to open source")
	}
	defer srcFile.Close()

	// ## Get source info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return MapError(err, src, "failed to stat source")
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
					Log.WithField("offset", startOffset).Info("Resuming copy at offset")
				} else if dstInfo.Size() == totalBytes {
					// Leaning on hash results to know if a file has been fully transferred
					Log.WithFields(logrus.Fields{
						"path": dst,
						"size": totalBytes,
					}).Info("File sizes match, verifying hashes...")
					srcHash, err := CalculateChecksum(c.srcFs, src, opts.HashAlgorithm)
					if err != nil {
						return MapError(err, src, "failed to calculate source hash")
					}
					dstHash, err := CalculateChecksum(c.dstFs, dst, opts.HashAlgorithm)
					if err != nil {
						return MapError(err, dst, "failed to calculate destination hash")
					}

					if srcHash == dstHash {
						Log.WithField("path", dst).Info("Hashes match, file already fully copied")
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
					return &models.CopyError{Code: models.ErrCodeChecksumMismatch, Message: "destination exists with same size but different hash", Path: dst}
				} else if dstInfo.Size() > totalBytes {
					return &models.CopyError{Code: models.ErrCodeUnknown, Message: "destination exists and is larger than source", Path: dst}
				} else if !opts.Resume {
					return &models.CopyError{Code: models.ErrCodeUnknown, Message: "destination exists and overwrite is false", Path: dst}
				}
			}
		}
	}

	// ## Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := c.dstFs.MkdirAll(destDir, 0755); err != nil {
		return MapError(err, destDir, "failed to create destination directory")
	}

	// ## Open/Create destination file
	var dstFile afero.File
	if isAppending {
		// Open for appending/resume
		dstFile, err = c.dstFs.OpenFile(dst, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return MapError(err, dst, "failed to open destination for resume")
		}

		// Seek source file to startOffset
		seeker, ok := srcFile.(io.Seeker)
		if !ok {
			return &models.CopyError{Code: models.ErrCodeUnknown, Message: "source file does not support seeking", Path: src}
		}
		if _, err := seeker.Seek(startOffset, io.SeekStart); err != nil {
			return MapError(err, src, "failed to seek source file")
		}
	} else {
		// New file or overwrite
		// If resume is true and file exists, it was likely handled above.
		// If it reaches here and exists, it means either:
		// 1. Resume is false and file exists (must check Overwrite)
		// 2. Resume is true but file size check failed (must check Overwrite)
		if exists, _ := afero.Exists(c.dstFs, dst); exists && !opts.Overwrite {
			return &models.CopyError{Code: models.ErrCodeUnknown, Message: "destination exists and overwrite is false", Path: dst}
		}

		dstFile, err = c.dstFs.Create(dst)
		if err != nil {
			return MapError(err, dst, "failed to create destination")
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
		return MapError(err, dst, "copy failed")
	}

	// ## Verification Logic: If we resumed, we MUST re-hash the source file fully
	if opts.CalculateHash {
		if startOffset > 0 {
			Log.WithField("path", src).Info("Performing mandatory full re-hash for resumed file")
			fullSourceHash, err := CalculateChecksum(c.srcFs, src, opts.HashAlgorithm)
			if err != nil {
				return MapError(err, src, "failed to calculate full source hash")
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
			return &models.CopyError{Code: models.ErrCodeChecksumMismatch, Message: fmt.Sprintf("checksum mismatch after copy: source %s, dest %s", hashResult, destHashResult), Path: dst}
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

// HashWithProgress calculates hashes for source and destination sequentially and streams progress.
func (c *Copier) HashWithProgress(ctx context.Context, src string, dst string, opts CopyOptions, progressChan chan<- Progress) (string, string, error) {
	if progressChan != nil {
		defer close(progressChan)
	}

	srcFile, err := c.srcFs.Open(src)
	if err != nil {
		return "", "", MapError(err, src, "failed to open source for hashing")
	}
	defer srcFile.Close()

	dstFile, err := c.dstFs.Open(dst)
	if err != nil {
		return "", "", MapError(err, dst, "failed to open destination for hashing")
	}
	defer dstFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return "", "", MapError(err, src, "failed to stat source for hashing")
	}
	totalBytes := srcInfo.Size()

	srcHasher, err := NewHasher(opts.HashAlgorithm)
	if err != nil {
		return "", "", err
	}
	dstHasher, err := NewHasher(opts.HashAlgorithm)
	if err != nil {
		return "", "", err
	}

	startTime := time.Now()

	// Hash source
	srcProxy := &ReaderProxy{
		Reader:  srcFile,
		Total:   totalBytes * 2,
		Context: ctx,
		ProgressCtx: &ProgressContext{
			Channel:     progressChan,
			StartTime:   startTime,
			BytesRead:   0,
			TotalOffset: 0,
		},
	}
	
	buf := make([]byte, opts.BufferSize)
	if len(buf) == 0 {
		buf = make([]byte, 1024*1024)
	}

	if _, err := io.CopyBuffer(srcHasher, srcProxy, buf); err != nil {
		return "", "", MapError(err, src, "failed to hash source")
	}
	srcHashResult := fmt.Sprintf("%x", srcHasher.Sum(nil))

	// Hash destination
	dstProxy := &ReaderProxy{
		Reader:  dstFile,
		Total:   totalBytes * 2,
		Context: ctx,
		ProgressCtx: &ProgressContext{
			Channel:     progressChan,
			StartTime:   startTime,
			BytesRead:   totalBytes,
			TotalOffset: 0,
		},
	}
	
	if _, err := io.CopyBuffer(dstHasher, dstProxy, buf); err != nil {
		return "", "", MapError(err, dst, "failed to hash destination")
	}
	dstHashResult := fmt.Sprintf("%x", dstHasher.Sum(nil))
	
	// Send final progress
	if progressChan != nil {
		progressChan <- Progress{
			BytesCopied: totalBytes * 2,
			TotalBytes:  totalBytes * 2,
			Speed:       float64(totalBytes*2) / time.Since(startTime).Seconds(),
			SourceHash:  srcHashResult,
			DestHash:    dstHashResult,
		}
	}

	if srcHashResult != dstHashResult {
		return srcHashResult, dstHashResult, &models.CopyError{Code: models.ErrCodeChecksumMismatch, Message: fmt.Sprintf("checksum mismatch during end check: source %s, dest %s", srcHashResult, dstHashResult), Path: dst}
	}

	return srcHashResult, dstHashResult, nil
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
