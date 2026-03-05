package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TransferService orchestrates transfer jobs
type TransferService struct {
	queue           *core.TransferQueue
	copier          *core.Copier
	scanner         *core.Scanner
	fs              afero.Fs
	settingsService *SettingsService
	ctx             context.Context // Wails runtime context
	running         bool
	cancel          context.CancelFunc
}

// NewTransferService creates a new TransferService
func NewTransferService(settings *SettingsService) *TransferService {
	// Use OS filesystem for actual operations
	fs := afero.NewOsFs()

	return &TransferService{
		queue:           core.NewTransferQueue(),
		copier:          core.NewCopier(fs, fs),
		scanner:         core.NewScanner(fs),
		fs:              fs,
		settingsService: settings,
	}
}

// SetContext sets the Wails context for event emitting
func (s *TransferService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// AddTransfer adds a new transfer job to the queue
func (s *TransferService) AddTransfer(sources []string, dest string, overwrite bool) (string, error) {
	core.Log.WithFields(logrus.Fields{
		"sources": sources,
		"dest":    dest,
	}).Debug("AddTransfer request received")
	// Scan sources
	files, _, totalSize, err := s.scanner.Scan(sources, dest)
	if err != nil {
		core.Log.WithError(err).Error("Scanner.Scan failed")
		return "", err
	}
	core.Log.WithFields(logrus.Fields{
		"fileCount": len(files),
		"totalSize": totalSize,
	}).Info("Sources scanned")

	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	// Set JobID on all files for easier tracking in frontend
	for i := range files {
		files[i].JobID = jobID
	}

	job := &models.TransferJob{
		ID:          jobID,
		Sources:     sources,
		Destination: dest,
		Overwrite:   overwrite,
		Status:      models.StatusPending,
		TotalFiles:  int64(len(files)),
		TotalBytes:  totalSize,
		Files:       files,
		CreatedAt:   time.Now().UnixMilli(),
	}

	s.queue.Add(job)
	s.emitQueueUpdate()
	core.Log.WithField("jobId", job.ID).Debug("Job added to queue")
	return job.ID, nil
}

// StartQueue starts processing the queue
func (s *TransferService) StartQueue() {
	core.Log.Debug("StartQueue called")
	if s.running {
		core.Log.Debug("StartQueue: already running")
		return
	}
	s.running = true
	go s.processQueue()
}

func (s *TransferService) processQueue() {
	core.Log.Debug("processQueue started")
	defer func() {
		core.Log.Debug("processQueue stopping")
		s.running = false
	}()

	for {
		// Get next pending job (simple Peek for now, assuming only one consumer)
		job := s.queue.Peek()
		if job == nil {
			core.Log.Debug("processQueue: Queue empty")
			return // Queue empty
		}

		if !s.processJob(job) {
			return // Cancelled or paused
		}
	}
}

// processJob handles the transfer of a single job
func (s *TransferService) processJob(job *models.TransferJob) bool {
	core.Log.WithFields(logrus.Fields{
		"jobId":     job.ID,
		"fileCount": len(job.Files),
	}).Info("Processing transfer job")

	// Update job status
	job.Status = models.StatusInProgress
	job.StartedAt = time.Now().UnixMilli()
	s.emitQueueUpdate()

	// Prepare for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	defer cancel()

	// Process files
	for i := range job.Files {
		file := job.Files[i]
		if file.Status == models.StatusSuccess || file.Status == models.StatusSkipped {
			continue
		}

		if err := s.processFile(ctx, job, file); err != nil {
			// Check if we should stop processing the entire job
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
				if job.Status == models.StatusPaused {
					s.emitQueueUpdate()
					return false // Pause requested, stop queue processing
				}
				// Job was canceled, finalize status and continue to next job
				job.Status = models.StatusFailed
				job.Error = "cancelled"
				job.CompletedAt = time.Now().UnixMilli()
				s.emitQueueUpdate()
				return true // Continue to next job in queue
			}
		}
	}

	// Finalize job status
	s.finalizeJobStatus(job)
	return true
}

// processFile handles copying a single file within a job
func (s *TransferService) processFile(ctx context.Context, job *models.TransferJob, file *models.FileInfo) error {
	core.Log.WithFields(logrus.Fields{
		"jobId": job.ID,
		"file":  file.Name,
	}).Debug("Processing file")

	file.Status = models.StatusInProgress
	file.ErrorMessage = "in progress"
	s.emitFileUpdate(file)
	s.emitQueueUpdate()

	settings := s.settingsService.Get()
	bufferSize := settings.BufferSize
	if bufferSize == 0 {
		bufferSize = 1024 * 1024
	}

	opts := core.CopyOptions{
		BufferSize:    bufferSize,
		CalculateHash: true,
		HashAlgorithm: core.HashAlgorithm(settings.HashAlgorithm),
		Overwrite:     job.Overwrite,
		Resume:        true,
		PreservePerms: true,
		PreserveTimes: true,
	}

	progressChan := make(chan core.Progress)
	errChan := make(chan error)

	go func() {
		errChan <- s.copier.CopyWithProgress(ctx, file.SourcePath, file.DestPath, opts, progressChan)
	}()

	// Handle progress updates
	var lastUpdate time.Time
	var lastProgress core.Progress

	for p := range progressChan {
		file.BytesCopied = p.BytesCopied
		lastProgress = p

		if time.Since(lastUpdate) > 200*time.Millisecond {
			s.updateJobProgress(job)
			s.emitProgress(job, p)
			s.emitFileUpdate(file)
			s.emitQueueUpdate()
			lastUpdate = time.Now()
		}
	}

	err := <-errChan
	s.handleCopyResult(file, err, lastProgress)
	s.updateJobProgress(job)
	s.emitFileUpdate(file)
	s.emitQueueUpdate()

	return err
}

func (s *TransferService) updateJobProgress(job *models.TransferJob) {
	var totalCopied int64
	for _, f := range job.Files {
		totalCopied += f.BytesCopied
	}
	job.BytesCopied = totalCopied
}

func (s *TransferService) handleCopyResult(file *models.FileInfo, err error, lastProgress core.Progress) {
	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
			file.Status = models.StatusPaused
			file.ErrorMessage = "paused"
		} else {
			file.Status = models.StatusFailed
			file.ErrorMessage = err.Error()
		}
	} else {
		file.Status = models.StatusSuccess
		file.ErrorMessage = "success"
		file.BytesCopied = file.Size
		file.SourceHash = lastProgress.SourceHash
		file.DestHash = lastProgress.DestHash
	}
}

func (s *TransferService) finalizeJobStatus(job *models.TransferJob) {
	job.Status = models.StatusSuccess
	for _, file := range job.Files {
		if file.Status == models.StatusFailed {
			job.Status = models.StatusFailed
			break
		}
	}
	job.CompletedAt = time.Now().UnixMilli()
	s.emitQueueUpdate()
}

func (s *TransferService) emitQueueUpdate() {
	if s.ctx == nil {
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

func (s *TransferService) Pause(jobID string) {
	core.Log.WithField("jobId", jobID).Info("Pause called for job")
	job := s.findJob(jobID)
	if job == nil {
		return
	}

	switch job.Status {
	case models.StatusInProgress:
		s.pauseActiveJob(job)
	case models.StatusPending:
		s.pausePendingJob(job)
	default:
		core.Log.WithFields(logrus.Fields{
			"jobId":  job.ID,
			"status": job.Status,
		}).Warn("Cannot pause job in status")
	}
}

func (s *TransferService) findJob(jobID string) *models.TransferJob {
	if jobID == "" {
		return s.queue.Peek()
	}
	for _, j := range s.queue.GetAll() {
		if j.ID == jobID {
			return j
		}
	}
	return nil
}

func (s *TransferService) pauseActiveJob(job *models.TransferJob) {
	job.Status = models.StatusPaused
	if s.cancel != nil {
		s.cancel()
	}
	for i := range job.Files {
		if job.Files[i].Status == models.StatusInProgress {
			job.Files[i].Status = models.StatusPaused
			job.Files[i].ErrorMessage = "paused"
		}
	}
	s.emitQueueUpdate()
}

func (s *TransferService) pausePendingJob(job *models.TransferJob) {
	job.Status = models.StatusPaused
	s.emitQueueUpdate()
}

func (s *TransferService) Resume(jobID string) {
	core.Log.WithField("jobId", jobID).Info("Resume called for job")
	jobs := s.queue.GetAll()
	resumed := false

	for _, job := range jobs {
		// If jobID is provided, only resume that specific job
		if jobID != "" && job.ID != jobID {
			continue
		}

		if s.tryResumeJob(job) {
			resumed = true
		}
	}

	if resumed {
		s.emitQueueUpdate()
		s.StartQueue()
	}
}

func (s *TransferService) tryResumeJob(job *models.TransferJob) bool {
	if job.Status != models.StatusPaused && job.Status != models.StatusFailed {
		return false
	}

	hasRemaining := false
	for _, file := range job.Files {
		if file.Status != models.StatusSuccess && file.Status != models.StatusSkipped {
			hasRemaining = true
			break
		}
	}
	if !hasRemaining {
		return false
	}

	// Reset job
	job.Status = models.StatusPending
	job.Error = ""

	// Reset interrupted files
	for _, file := range job.Files {
		if file.Status == models.StatusInProgress || file.Status == models.StatusPaused || file.Status == models.StatusFailed {
			file.Status = models.StatusPending
			file.ErrorMessage = ""
		}
	}
	return true
}

func (s *TransferService) Cancel(jobID string) {
	core.Log.WithField("jobId", jobID).Info("Cancel called for job")
	job := s.findJobToCancel(jobID)
	if job != nil {
		s.cancelSpecificJob(job)
		// Restart queue processing in case it was stopped due to this job being paused
		s.StartQueue()
	}
}

func (s *TransferService) findJobToCancel(jobID string) *models.TransferJob {
	if jobID != "" {
		for _, j := range s.queue.GetAll() {
			if j.ID == jobID {
				return j
			}
		}
		return nil
	}
	// Default legacy behavior
	for _, j := range s.queue.GetAll() {
		if j.Status == models.StatusInProgress || j.Status == models.StatusPaused || j.Status == models.StatusPending {
			return j
		}
	}
	return nil
}

func (s *TransferService) cancelSpecificJob(job *models.TransferJob) {
	oldStatus := job.Status
	job.Status = models.StatusFailed
	job.Error = "cancelled"
	job.CompletedAt = time.Now().UnixMilli()

	if oldStatus == models.StatusInProgress && s.cancel != nil {
		s.cancel()
	}
	for i := range job.Files {
		status := job.Files[i].Status
		if status == models.StatusInProgress || status == models.StatusPending || status == models.StatusPaused {
			job.Files[i].Status = models.StatusFailed
			job.Files[i].ErrorMessage = "cancelled"
		}
	}
	s.emitQueueUpdate()
}
