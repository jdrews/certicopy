package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Copier interface for file operations
type Copier interface {
	Copy(src string, dst string, opts core.CopyOptions) error
	CopyWithProgress(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error
	HashWithProgress(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) (string, string, error)
}

// Scanner interface for filesystem scanning
type Scanner interface {
	Scan(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error)
}

// TransferService orchestrates transfer jobs
type TransferService struct {
	queue           *core.TransferQueue
	copier          Copier
	scanner         Scanner
	fs              afero.Fs
	settingsService *SettingsService
	ctx             context.Context // Wails runtime context
	running         bool
	cancel          context.CancelFunc
	jobAdded        chan struct{}
	mutex           sync.RWMutex
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
		jobAdded:        make(chan struct{}, 10),
	}
}

// NewTransferServiceWithDeps creates a new TransferService with specific dependencies for testing
func NewTransferServiceWithDeps(
	queue *core.TransferQueue,
	copier Copier,
	scanner Scanner,
	fs afero.Fs,
	settings *SettingsService,
) *TransferService {
	return &TransferService{
		queue:           queue,
		copier:          copier,
		scanner:         scanner,
		fs:              fs,
		settingsService: settings,
		jobAdded:        make(chan struct{}, 10),
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

	// Normalize destination to absolute path
	if absDest, err := filepath.Abs(dest); err == nil {
		dest = absDest
	}

	// Normalize sources to absolute paths
	for i, src := range sources {
		if absSrc, err := filepath.Abs(src); err == nil {
			sources[i] = absSrc
		}
	}

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

	// Start queue processing
	s.StartQueue()

	// Signal that a job was added
	select {
	case s.jobAdded <- struct{}{}:
	default:
	}

	return job.ID, nil
}

// StartQueue starts processing the queue
func (s *TransferService) StartQueue() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	core.Log.Debug("StartQueue called")

	// Always try to signal the worker in case it's waiting
	select {
	case s.jobAdded <- struct{}{}:
	default:
	}

	if s.running {
		core.Log.Debug("StartQueue: already running")
		return
	}
	s.running = true
	go s.processQueue()
}

func (s *TransferService) processQueue() {
	core.Log.Info("processQueue started")
	defer func() {
		s.mutex.Lock()
		s.running = false
		s.mutex.Unlock()
		core.Log.Info("processQueue stopping")
	}()

	for {
		// Get next pending job under the service mutex so we don't race on job.Status
		job := s.findPendingJob()
		if job == nil {
			core.Log.Info("processQueue: Queue empty, waiting for signal")
			// Wait for a signal that a job was added
			select {
			case <-s.jobAdded:
				core.Log.Info("processQueue: Signal received")
				continue
			case <-time.After(10 * time.Second): // Timeout to allow goroutine cleanup
				return
			}
		}

		// Go ahead with processing the job.
		// processJob returns true when a job has finished (success/failure)
		// and the queue should proceed to the next job. It returns false
		// if the job was paused or interrupted, and we should just cycle.
		if !s.processJob(job) {
			core.Log.Info("processQueue: Job paused or interrupted, continuing loop")
			continue
		}
	}
}

// findPendingJob returns the first job with status Pending or InProgress,
// reading job.Status under s.mutex to avoid races with control methods.
func (s *TransferService) findPendingJob() *models.TransferJob {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, j := range s.queue.GetAll() {
		if j.Status == models.StatusPending || j.Status == models.StatusInProgress {
			return j
		}
	}
	return nil
}

// processJob handles the transfer of a single job
func (s *TransferService) processJob(job *models.TransferJob) bool {
	s.mutex.Lock()
	fileCountLogging := len(job.Files)
	s.mutex.Unlock()
	core.Log.WithFields(logrus.Fields{
		"jobId":     job.ID,
		"fileCount": fileCountLogging,
	}).Debug("processJob: Starting")

	// Update job status
	s.mutex.Lock()
	job.Status = models.StatusInProgress
	job.StartedAt = time.Now().UnixMilli()
	s.mutex.Unlock()
	s.emitQueueUpdate()

	// Prepare for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()
	defer func() {
		s.mutex.Lock()
		s.cancel = nil
		s.mutex.Unlock()
		cancel()
	}()

	consecutiveSysFailures := 0

	// Process files
	s.mutex.Lock()
	fileCount := len(job.Files)
	s.mutex.Unlock()
	
	for i := 0; i < fileCount; i++ {
		s.mutex.Lock()
		file := job.Files[i]
		status := file.Status
		s.mutex.Unlock()
		if status == models.StatusSuccess || status == models.StatusSkipped {
			continue
		}

		if err := s.processFileWithRetry(ctx, job, file); err != nil {
			// Track systemic failures
			var copyErr *models.CopyError
			if errors.As(err, &copyErr) && !copyErr.IsAutoRetryable() && copyErr.Code != models.ErrCodeUnknown {
				consecutiveSysFailures++
			} else {
				consecutiveSysFailures = 0
			}

			if consecutiveSysFailures >= 3 { // Max consecutive failures
				core.Log.WithField("jobId", job.ID).Warn("Circuit breaker triggered, pausing job")
				s.mutex.Lock()
				job.Status = models.StatusFailed
				job.Error = "Auto-paused due to consecutive systemic failures"
				if copyErr != nil {
					job.ErrorCode = string(copyErr.Code)
				}
				s.mutex.Unlock()
				s.pauseActiveJob(job)
				return false
			}

			// Check if we should stop processing the entire job
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
				s.mutex.Lock()
				status := job.Status
				s.mutex.Unlock()
				core.Log.WithFields(logrus.Fields{
					"jobId":  job.ID,
					"status": status,
				}).Debug("processJob: Context cancelled, checking job status")

				if status == models.StatusPaused || status == models.StatusPending {
					s.emitQueueUpdate()
					return false // Stop current processing loop, let queue restart/continue
				}
				// Job was canceled terminaly
				s.mutex.Lock()
				job.Status = models.StatusFailed
				job.Error = "cancelled"
				job.CompletedAt = time.Now().UnixMilli()
				s.mutex.Unlock()
				s.emitQueueUpdate()
				return true // Continue to next job in queue
			}
		} else {
			consecutiveSysFailures = 0 // Reset on success
		}
	}

	// Post-processing: End Hash Check
	settings := s.settingsService.Get()
	if settings.EndCheck {
		hasSuccessFiles := false
		s.mutex.Lock()
		for _, file := range job.Files {
			if file.Status == models.StatusSuccess {
				hasSuccessFiles = true
				break
			}
		}
		s.mutex.Unlock()

		if hasSuccessFiles && s.processHashPhase(ctx, job) == false {
			return false // Cancelled or paused during hashing
		}
	}

	// Finalize job status
	s.finalizeJobStatus(job)
	core.Log.WithField("jobId", job.ID).Info("processJob: Finished successfully")
	return true
}

// processHashPhase performs a full hash re-check of successfully transferred files
func (s *TransferService) processHashPhase(ctx context.Context, job *models.TransferJob) bool {
	core.Log.WithField("jobId", job.ID).Info("Starting end hash check phase")

	s.mutex.Lock()
	job.Status = models.StatusHashing
	job.BytesCopied = 0

	// Recalculate total bytes for hashing logic (each successful file will be hashed twice, src and dst)
	var hashTotalBytes int64
	for _, file := range job.Files {
		if file.Status == models.StatusSuccess {
			hashTotalBytes += file.Size * 2
			// Clear hashes to indicate they are being recalculated
			file.SourceHash = ""
			file.DestHash = ""
			file.BytesCopied = 0
			file.Status = models.StatusHashing
			file.ErrorMessage = "verifying integrity..."
			// we can emit later, but emit doesn't block much if it uses runtime
			s.emitFileUpdate(file)
		}
	}
	job.TotalBytes = hashTotalBytes
	s.mutex.Unlock()
	s.emitQueueUpdate()

	settings := s.settingsService.Get()
	for i := range job.Files {
		s.mutex.Lock()
		file := job.Files[i]
		fileStatus := file.Status
		s.mutex.Unlock()
		if fileStatus != models.StatusHashing {
			continue // Only re-hash files that successfully copied
		}

		opts := core.CopyOptions{
			BufferSize:    settings.BufferSize,
			HashAlgorithm: core.HashAlgorithm(settings.HashAlgorithm),
		}

		progressChan := make(chan core.Progress)
		errChan := make(chan error)
		s.mutex.Lock()
		file.ErrorMessage = "hashing..." // Clear the previous "transferred" message
		s.mutex.Unlock()
		s.emitFileUpdate(file)

		go func() {
			srcHash, dstHash, err := s.copier.HashWithProgress(ctx, file.SourcePath, file.DestPath, opts, progressChan)
			if err == nil {
				file.SourceHash = srcHash
				file.DestHash = dstHash
			}
			errChan <- err
		}()

		var lastUpdate time.Time

		for p := range progressChan {
			// p.BytesCopied contains BytesRead for this file (up to file.Size * 2)
			s.mutex.Lock()
			file.BytesCopied = p.BytesCopied
			s.mutex.Unlock()

			if time.Since(lastUpdate) > 200*time.Millisecond {
				s.updateJobProgress(job)
				s.emitProgress(job, p)
				s.emitFileUpdate(file)
				s.emitQueueUpdate()
				lastUpdate = time.Now()
			}
		}

		err := <-errChan
		s.mutex.Lock()
		if err != nil {
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
				file.Status = models.StatusPaused
				job.Status = models.StatusPaused
				s.mutex.Unlock()
				s.emitFileUpdate(file)
				s.emitQueueUpdate()
				return false
			}
			file.Status = models.StatusFailed
			file.ErrorMessage = fmt.Sprintf("hash verify failed: %v", err)
			file.EndHashVerified = false
		} else {
			file.Status = models.StatusSuccess // switch back to success
			file.BytesCopied = file.Size * 2   // leave it at total bytes hashed for UI until we revert
			file.EndHashVerified = true
			file.ErrorMessage = "success: integrity verified"
		}
		s.mutex.Unlock()

		s.updateJobProgress(job)
		s.emitFileUpdate(file)
		s.emitQueueUpdate()
	}

	// Revert job tracking back to normal size so "Complete" shows correct total bytes
	var originalTotalBytes int64
	s.mutex.Lock()
	for _, file := range job.Files {
		// We use file.Size, the true original size
		originalTotalBytes += file.Size
		// Revert successful files to correct BytesCopied
		if file.Status == models.StatusSuccess {
			file.BytesCopied = file.Size
		}
	}
	job.TotalBytes = originalTotalBytes
	job.BytesCopied = originalTotalBytes // Reset so percentage is 100%
	s.mutex.Unlock()
	s.emitQueueUpdate()

	return true
}

// processFileWithRetry handles retries with exponential backoff for a file
func (s *TransferService) processFileWithRetry(ctx context.Context, job *models.TransferJob, file *models.FileInfo) error {
	maxRetries := 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := s.processFile(ctx, job, file)
		if err == nil {
			return nil // Success
		}

		var copyErr *models.CopyError
		if errors.As(err, &copyErr) && !copyErr.IsAutoRetryable() {
			return err // Terminal error based on code
		}

		if attempt == maxRetries {
			return err // Max retries reached
		}

		// Exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(baseDelay * time.Duration(1<<attempt)):
			// Retry
		}
	}
	return nil
}

// processFile handles copying a single file within a job
func (s *TransferService) processFile(ctx context.Context, job *models.TransferJob, file *models.FileInfo) error {
	core.Log.WithFields(logrus.Fields{
		"jobId": job.ID,
		"file":  file.Name,
	}).Debug("Processing file")

	s.mutex.Lock()
	file.Status = models.StatusInProgress
	file.ErrorMessage = "in progress"
	s.mutex.Unlock()
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
		s.mutex.Lock()
		file.BytesCopied = p.BytesCopied
		s.mutex.Unlock()
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var totalCopied int64
	for _, f := range job.Files {
		totalCopied += f.BytesCopied
	}
	job.BytesCopied = totalCopied
}

func (s *TransferService) handleCopyResult(file *models.FileInfo, err error, lastProgress core.Progress) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
			file.Status = models.StatusPaused
			file.ErrorMessage = "paused"
		} else {
			file.Status = models.StatusFailed
			file.ErrorMessage = err.Error()
			var copyErr *models.CopyError
			if errors.As(err, &copyErr) {
				file.ErrorCode = string(copyErr.Code)
				file.ErrorMessage = copyErr.Message
			}
		}
	} else {
		file.Status = models.StatusSuccess
		file.ErrorMessage = "transferred"
		file.BytesCopied = file.Size
		file.TransferCompleted = true
		file.SourceHash = lastProgress.SourceHash
		file.DestHash = lastProgress.DestHash
	}
}

func (s *TransferService) finalizeJobStatus(job *models.TransferJob) {
	s.mutex.Lock()
	job.Status = models.StatusSuccess
	for _, file := range job.Files {
		if file.Status == models.StatusFailed {
			job.Status = models.StatusFailed
			break
		}
	}
	job.CompletedAt = time.Now().UnixMilli()
	s.mutex.Unlock()
	s.emitQueueUpdate()
}

func (s *TransferService) emitQueueUpdate() {
	if s.ctx == nil {
		return
	}
	// RLock: we only read job/file fields to clone them for the UI event.
	s.mutex.RLock()
	data := s.queue.GetAllClones()
	s.mutex.RUnlock()
	runtime.EventsEmit(s.ctx, "queue:updated", data)
}

func (s *TransferService) emitFileUpdate(file *models.FileInfo) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "file:updated", file)
	}
}

func (s *TransferService) emitProgress(job *models.TransferJob, p core.Progress) {
	if s.ctx != nil {
		s.mutex.RLock()
		jobId := job.ID
		bytesCopied := job.BytesCopied
		totalBytes := job.TotalBytes
		s.mutex.RUnlock()
		
		runtime.EventsEmit(s.ctx, "transfer:progress", map[string]interface{}{
			"jobId":       jobId,
			"bytesCopied": bytesCopied,
			"totalBytes":  totalBytes,
			"speed":       p.Speed,
		})
	}
}

func (s *TransferService) GetQueue() []*models.TransferJob {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.queue.GetAllClones()
}

func (s *TransferService) Pause(jobID string) {
	core.Log.WithField("jobId", jobID).Info("Pause called for job")
	job := s.findJob(jobID)
	if job == nil {
		return
	}

	s.mutex.Lock()
	status := job.Status
	s.mutex.Unlock()

	switch status {
	case models.StatusInProgress:
		s.pauseActiveJob(job)
	case models.StatusPending:
		s.pausePendingJob(job)
	default:
		core.Log.WithFields(logrus.Fields{
			"jobId":  job.ID,
			"status": status,
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
	s.mutex.Lock()
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
	s.mutex.Unlock()
	s.emitQueueUpdate()
}

func (s *TransferService) pausePendingJob(job *models.TransferJob) {
	s.mutex.Lock()
	job.Status = models.StatusPaused
	s.mutex.Unlock()
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
		core.Log.WithField("jobId", jobID).Debug("Resume: Job successfully resumed and signaled")
		s.emitQueueUpdate()
		s.StartQueue()
		// Signal that a job was resumed
		select {
		case s.jobAdded <- struct{}{}:
		default:
		}
	}
}

func (s *TransferService) tryResumeJob(job *models.TransferJob) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
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
	// Default legacy behavior - find first active job
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, j := range s.queue.GetAll() {
		if j.Status == models.StatusInProgress || j.Status == models.StatusPaused || j.Status == models.StatusPending {
			return j
		}
	}
	return nil
}

func (s *TransferService) cancelSpecificJob(job *models.TransferJob) {
	s.mutex.Lock()
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
	s.mutex.Unlock()
	s.emitQueueUpdate()
}

// RemoveFileFromJob completely removes a file from a job by source path
func (s *TransferService) RemoveFileFromJob(jobID string, sourcePath string) {
	job := s.findJob(jobID)
	if job == nil {
		return
	}

	s.mutex.Lock()
	filtered := make([]*models.FileInfo, 0, len(job.Files))
	for _, f := range job.Files {
		if f.SourcePath != sourcePath {
			filtered = append(filtered, f)
		}
	}
	// Update counts
	job.Files = filtered
	job.TotalFiles = int64(len(filtered))
	var newTotalSize int64
	for _, f := range filtered {
		newTotalSize += f.Size
	}
	job.TotalBytes = newTotalSize
	s.mutex.Unlock()
	s.updateJobProgress(job)

	s.emitQueueUpdate()
}
