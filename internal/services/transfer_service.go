package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/spf13/afero"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TransferService orchestrates transfer jobs
type TransferService struct {
	queue   *core.TransferQueue
	copier  *core.Copier
	scanner *core.Scanner
	fs      afero.Fs
	ctx     context.Context // Wails runtime context
	running bool
	cancel  context.CancelFunc
}

// NewTransferService creates a new TransferService
func NewTransferService() *TransferService {
	// Use OS filesystem for actual operations
	fs := afero.NewOsFs()

	return &TransferService{
		queue:   core.NewTransferQueue(),
		copier:  core.NewCopier(fs, fs),
		scanner: core.NewScanner(fs),
		fs:      fs,
	}
}

// SetContext sets the Wails context for event emitting
func (s *TransferService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// AddTransfer adds a new transfer job to the queue
func (s *TransferService) AddTransfer(sources []string, dest string) (string, error) {
	fmt.Printf("AddTransfer called with sources: %v, dest: %s\n", sources, dest)
	// Scan sources
	files, _, totalSize, err := s.scanner.Scan(sources, dest)
	if err != nil {
		fmt.Printf("Scanner.Scan failed: %v\n", err)
		return "", err
	}
	fmt.Printf("Scanner found %d files, total size: %d\n", len(files), totalSize)

	job := &models.TransferJob{
		ID:          fmt.Sprintf("job_%d", time.Now().UnixNano()),
		Sources:     sources,
		Destination: dest,
		Status:      models.StatusPending,
		TotalFiles:  int64(len(files)),
		TotalBytes:  totalSize,
		Files:       files,
		CreatedAt:   time.Now(),
	}

	s.queue.Add(job)
	s.emitQueueUpdate()
	fmt.Println("Job added to queue and update emitted")
	return job.ID, nil
}

// StartQueue starts processing the queue
func (s *TransferService) StartQueue() {
	fmt.Println("StartQueue called")
	if s.running {
		fmt.Println("StartQueue: already running")
		return
	}
	s.running = true
	go s.processQueue()
}

func (s *TransferService) processQueue() {
	fmt.Println("processQueue started")
	defer func() {
		fmt.Println("processQueue stopping")
		s.running = false
	}()

	for {
		// Get next pending job (simple Peek for now, assuming only one consumer)
		job := s.queue.Peek()
		if job == nil {
			fmt.Println("processQueue: Queue empty")
			return // Queue empty
		}
		fmt.Printf("processQueue: Processing job %s with %d files\n", job.ID, len(job.Files))

		// Update job status
		job.Status = models.StatusInProgress
		job.StartedAt = time.Now()
		s.emitQueueUpdate()
		// Prepare for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel

		// Process files
		for i := range job.Files {
			file := job.Files[i] // Pointer to file in slice
			fmt.Printf("processQueue: Checking file %s (Status: %s)\n", file.Name, file.Status)

			// Check for cancellation
			select {
			case <-ctx.Done():
				job.Status = models.StatusFailed
				job.Error = "Cancelled"
				s.emitQueueUpdate()
				return
			default:
			}

			if file.Status == models.StatusSuccess || file.Status == models.StatusSkipped {
				continue
			}

			file.Status = models.StatusInProgress
			s.emitFileUpdate(file)
			s.emitQueueUpdate() // Update queue so UI shows in_progress state

			// Perform copy configuration
			opts := core.CopyOptions{
				BufferSize:    1024 * 1024, // 1MB default
				CalculateHash: true,
				HashAlgorithm: core.HashXXHash, // Default
				Overwrite:     true,            // Default to overwrite for now
				PreservePerms: true,
				PreserveTimes: true,
			}

			progressChan := make(chan core.Progress)

			// Run copy in goroutine to process progress updates
			errChan := make(chan error)
			go func() {
				fmt.Printf("Starting copy for file: %s\n", file.Name)
				defer close(errChan)
				errChan <- s.copier.CopyWithProgress(file.SourcePath, file.DestPath, opts, progressChan)
				fmt.Printf("Copy goroutine finished for file: %s\n", file.Name)
			}()

			// Listen for progress
			var lastUpdate time.Time
			var lastProgress core.Progress

			for p := range progressChan {
				file.BytesCopied = p.BytesCopied
				lastProgress = p

				// Throttle progress emissions to every 200ms
				if time.Since(lastUpdate) > 200*time.Millisecond {
					// Update job-level bytesCopied for overall progress
					var totalCopied int64
					for _, f := range job.Files {
						totalCopied += f.BytesCopied
					}
					job.BytesCopied = totalCopied

					s.emitProgress(job, p)
					s.emitFileUpdate(file)
					s.emitQueueUpdate()
					lastUpdate = time.Now()
				}
			}

			err := <-errChan
			if err != nil {
				fmt.Printf("Copy failed for %s: %v\n", file.Name, err)
				file.Status = models.StatusFailed
				file.ErrorMessage = err.Error()
			} else {
				fmt.Printf("Copy success for %s\n", file.Name)
				file.Status = models.StatusSuccess
				file.BytesCopied = file.Size // Ensure 100%
				file.SourceHash = lastProgress.SourceHash
				file.DestHash = lastProgress.DestHash
			}

			// Update job-level stats after each file
			var totalCopied int64
			for _, f := range job.Files {
				totalCopied += f.BytesCopied
			}
			job.BytesCopied = totalCopied

			s.emitFileUpdate(file)
			s.emitQueueUpdate() // Critical: emit queue update so UI refreshes all file statuses
		}

		// Determine final job status
		job.Status = models.StatusSuccess
		for _, file := range job.Files {
			if file.Status == models.StatusFailed {
				job.Status = models.StatusFailed
				break
			}
		}

		job.CompletedAt = time.Now()

		// Keep completed job in queue so sidebar shows it as done
		// Don't Pop - let user clear it manually
		s.emitQueueUpdate()
	}
}

func (s *TransferService) emitQueueUpdate() {
	if s.ctx == nil {
		fmt.Println("emitQueueUpdate: context is nil, cannot emit")
		return
	}
	runtime.EventsEmit(s.ctx, "queue:updated", s.queue.GetAll())
}

func (s *TransferService) emitFileUpdate(file *models.FileInfo) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "file:updated", file)
	}
}

func (s *TransferService) emitProgress(job *models.TransferJob, p core.Progress) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "transfer:progress", map[string]interface{}{
			"jobId":       job.ID,
			"bytesCopied": job.BytesCopied,
			"totalBytes":  job.TotalBytes,
			"speed":       p.Speed,
		})
	}
}

func (s *TransferService) GetQueue() []*models.TransferJob {
	return s.queue.GetAll()
}

func (s *TransferService) Pause() {
	// TODO: Implement pause
}

func (s *TransferService) Resume() {
	fmt.Println("Resume called")
	// Find jobs that were stopped/cancelled and have remaining files to copy
	jobs := s.queue.GetAll()
	resumed := false
	for _, job := range jobs {
		// Only resume jobs that are not currently running and have incomplete files
		if job.Status == models.StatusFailed || job.Status == models.StatusPending {
			hasRemaining := false
			for _, file := range job.Files {
				if file.Status != models.StatusSuccess && file.Status != models.StatusSkipped {
					hasRemaining = true
					break
				}
			}
			if !hasRemaining {
				continue
			}

			// Reset job to pending so processQueue will pick it up
			job.Status = models.StatusPending
			job.Error = ""

			// Reset any interrupted (in_progress) or failed files back to pending
			// so they get re-copied. Success/skipped files are preserved.
			for _, file := range job.Files {
				if file.Status == models.StatusInProgress || file.Status == models.StatusFailed {
					file.Status = models.StatusPending
					file.ErrorMessage = ""
					file.BytesCopied = 0
				}
			}

			resumed = true
		}
	}

	if resumed {
		s.emitQueueUpdate()
		s.StartQueue()
	}
}

func (s *TransferService) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}
