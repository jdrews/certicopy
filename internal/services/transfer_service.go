package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		CreatedAt:   time.Now().UnixMilli(),
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
		job.StartedAt = time.Now().UnixMilli()
		s.emitQueueUpdate()
		// Prepare for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel

		// Process files
		for i := range job.Files {
			file := job.Files[i] // Pointer to file in slice
			fmt.Printf("processQueue: Checking file %s (Status: %s)\n", file.Name, file.Status)

			select {
			case <-ctx.Done():
				// If Pause was called, job.Status will already be StatusPaused.
				// If Cancel was called, it'll still be "InProgress" or "Failed"
				if job.Status != models.StatusPaused {
					job.Status = models.StatusFailed
					job.Error = "cancelled"
				}
				s.emitQueueUpdate()
				return
			default:
			}

			if file.Status == models.StatusSuccess || file.Status == models.StatusSkipped {
				continue
			}

			file.Status = models.StatusInProgress
			file.ErrorMessage = "in progress" // Show user-friendly status
			s.emitFileUpdate(file)
			s.emitQueueUpdate() // Update queue so UI shows in_progress state

			// Perform copy configuration
			opts := core.CopyOptions{
				BufferSize:    1024 * 1024, // 1MB default
				CalculateHash: true,
				HashAlgorithm: core.HashXXHash, // Default
				Overwrite:     false,           // Don't overwrite when resuming
				Resume:        true,            // Always try to resume
				PreservePerms: true,
				PreserveTimes: true,
			}

			progressChan := make(chan core.Progress)

			// Run copy in goroutine to process progress updates
			errChan := make(chan error)
			go func() {
				fmt.Printf("Starting copy for file: %s\n", file.Name)
				defer close(errChan)
				errChan <- s.copier.CopyWithProgress(ctx, file.SourcePath, file.DestPath, opts, progressChan)
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
				// Use both errors.Is and string check for robustness against wrapped errors
				if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
					fmt.Printf("Copy paused for %s\n", file.Name)
					file.Status = models.StatusPaused
					file.ErrorMessage = "paused"
				} else {
					fmt.Printf("Copy failed for %s: %v\n", file.Name, err)
					file.Status = models.StatusFailed
					file.ErrorMessage = err.Error()
				}
			} else {
				fmt.Printf("Copy success for %s\n", file.Name)
				file.Status = models.StatusSuccess
				file.ErrorMessage = "success" // Explicitly show success
				file.BytesCopied = file.Size  // Ensure 100%
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

		job.CompletedAt = time.Now().UnixMilli()

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
	fmt.Println("Pause called")
	if s.cancel != nil {
		s.cancel()
		// Update status of active job and any in_progress files to "paused"
		job := s.queue.Peek()
		if job != nil && job.Status == models.StatusInProgress {
			job.Status = models.StatusPaused
			for i := range job.Files {
				if job.Files[i].Status == models.StatusInProgress {
					job.Files[i].Status = models.StatusPaused
					job.Files[i].ErrorMessage = "paused"
				}
			}
			s.emitQueueUpdate()
		}
	}
}

func (s *TransferService) Resume() {
	fmt.Println("Resume called")
	// Find jobs that were paused or failed and have remaining files to copy
	jobs := s.queue.GetAll()
	resumed := false
	for _, job := range jobs {
		// Only resume jobs that are not currently running and have incomplete files
		if job.Status == models.StatusPaused || job.Status == models.StatusFailed {
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

			// Reset any interrupted (paused/failed) files back to pending
			// but KEEP their BytesCopied so copier can resume
			for _, file := range job.Files {
				if file.Status == models.StatusInProgress || file.Status == models.StatusPaused || file.Status == models.StatusFailed {
					file.Status = models.StatusPending
					file.ErrorMessage = ""
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
	fmt.Println("Cancel called")
	if s.cancel != nil {
		s.cancel()
	}

	// Find the job to cancel. We don't use Peek() because it skips Paused jobs.
	var jobToCancel *models.TransferJob
	for _, job := range s.queue.GetAll() {
		if job.Status == models.StatusInProgress || job.Status == models.StatusPaused || job.Status == models.StatusPending {
			jobToCancel = job
			break
		}
	}

	if jobToCancel != nil {
		jobToCancel.Status = models.StatusFailed
		jobToCancel.Error = "cancelled"
		for i := range jobToCancel.Files {
			if jobToCancel.Files[i].Status == models.StatusInProgress ||
				jobToCancel.Files[i].Status == models.StatusPending ||
				jobToCancel.Files[i].Status == models.StatusPaused {
				jobToCancel.Files[i].Status = models.StatusFailed
				jobToCancel.Files[i].ErrorMessage = "cancelled"
			}
		}
		s.emitQueueUpdate()
	}
}
