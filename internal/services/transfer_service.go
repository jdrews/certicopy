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
	// Scan sources
	files, _, totalSize, err := s.scanner.Scan(sources, dest)
	if err != nil {
		return "", err
	}

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
	return job.ID, nil
}

// StartQueue starts processing the queue
func (s *TransferService) StartQueue() {
	if s.running {
		return
	}
	s.running = true
	go s.processQueue()
}

func (s *TransferService) processQueue() {
	defer func() { s.running = false }()

	for {
		// Get next pending job (simple Peek for now, assuming only one consumer)
		job := s.queue.Peek()
		if job == nil {
			return // Queue empty
		}

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
				defer close(errChan)
				// copier closes progressChan
				errChan <- s.copier.CopyWithProgress(file.SourcePath, file.DestPath, opts, progressChan)
			}()

			// Listen for progress
			var lastUpdate time.Time
			// var initialBytesCopied = job.BytesCopied // Snapshot at start of file // Not used in this version
			var lastProgress core.Progress // To capture the final progress value

			for p := range progressChan {
				// Update file progress
				file.BytesCopied = p.BytesCopied
				lastProgress = p // Capture the last progress update

				// Let's rely on emitting FILE progress and let UI update job bar?
				// Or emit job progress event with aggregated stats.
				// TODO: Decide on progress reporting strategy

				if time.Since(lastUpdate) > 500*time.Millisecond {
					s.emitProgress(job, p) // Emits current file progress
					lastUpdate = time.Now()
				}
			}

			err := <-errChan
			if err != nil {
				file.Status = models.StatusFailed
				file.ErrorMessage = err.Error()
			} else {
				file.Status = models.StatusSuccess
				// We need the hash from the copier progress (last value)
				// Ideally CopyWithProgress returns it or we capture it from channel
				// The channel is closed, so we might miss the very last value if loop exits?
				// No, range loop consumes all.
				// But we didn't save the progress value outside the loop.
				// Let's modify loop above to capture last p.
				file.SourceHash = lastProgress.SourceHash
				file.DestHash = lastProgress.DestHash
			}
			s.emitFileUpdate(file)
		}

		job.Status = models.StatusSuccess // Need to check if any failed
		// TODO: Check if any files failed, if so, set job.Status = models.StatusFailed
		for _, file := range job.Files {
			if file.Status == models.StatusFailed {
				job.Status = models.StatusFailed
				break
			}
		}

		job.CompletedAt = time.Now()

		// Remove from queue (Pop) after completion?
		// Or keep it as "completed" history?
		// For now, let's Pop it so queue only shows pending/active.
		s.queue.Pop()

		s.emitQueueUpdate()
	}
}

func (s *TransferService) emitQueueUpdate() {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "queue:updated", s.queue.GetAll())
	}
}

func (s *TransferService) emitFileUpdate(file *models.FileInfo) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "file:updated", file)
	}
}

func (s *TransferService) emitProgress(job *models.TransferJob, p core.Progress) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "transfer:progress", p)
	}
}

func (s *TransferService) GetQueue() []*models.TransferJob {
	return s.queue.GetAll()
}

func (s *TransferService) Pause() {
	// TODO: Implement pause
}

func (s *TransferService) Resume() {
	// TODO: Implement resume
}

func (s *TransferService) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}
