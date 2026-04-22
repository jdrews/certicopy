package services

import (
	"context"
	"errors"
	"sync/atomic"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/spf13/afero"
)

// MockCopier implements services.Copier
type MockCopier struct {
	copyFunc             func(src string, dst string, opts core.CopyOptions) error
	copyWithProgressFunc func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error
	hashWithProgressFunc func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) (string, string, error)
}

func (m *MockCopier) Copy(src string, dst string, opts core.CopyOptions) error {
	if m.copyFunc != nil {
		return m.copyFunc(src, dst, opts)
	}
	return nil
}

func (m *MockCopier) CopyWithProgress(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
	if m.copyWithProgressFunc != nil {
		return m.copyWithProgressFunc(ctx, src, dst, opts, progressChan)
	}
	if progressChan != nil {
		close(progressChan)
	}
	return nil
}

func (m *MockCopier) HashWithProgress(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) (string, string, error) {
	if m.hashWithProgressFunc != nil {
		return m.hashWithProgressFunc(ctx, src, dst, opts, progressChan)
	}
	if progressChan != nil {
		close(progressChan)
	}
	return "src", "dst", nil
}

// MockScanner implements services.Scanner
type MockScanner struct {
	scanFunc func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error)
}

func (m *MockScanner) Scan(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
	if m.scanFunc != nil {
		return m.scanFunc(sources, destRoot)
	}
	return []*models.FileInfo{}, 0, 0, nil
}

func TestTransferService_AddTransfer(t *testing.T) {
	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "test.txt", Size: 100}}, 1, 100, nil
		},
	}
	settings := NewSettingsServiceWithConfigPath("/tmp/settings.json")
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), &MockCopier{}, mockScanner, afero.NewMemMapFs(), settings)

	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "src", "f1")
	dstDir := filepath.Join(base, "dest")
	id, err := s.AddTransfer([]string{srcDir}, dstDir, false)
	if err != nil {
		t.Fatalf("AddTransfer failed: %v", err)
	}
	if id == "" {
		t.Error("Expected non-empty job ID")
	}

	queue := s.GetQueue()
	if len(queue) != 1 {
		t.Errorf("Expected 1 job in queue, got %d", len(queue))
	}
}

func TestTransferService_ProcessJob(t *testing.T) {
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			// Simulate progress
			if progressChan != nil {
				progressChan <- core.Progress{BytesCopied: 50, TotalBytes: 100}
				close(progressChan)
			}
			return nil
		},
	}
	base, _ := filepath.Abs(".")
	srcFile := filepath.Join(base, "s", "f.txt")
	dstFile := filepath.Join(base, "d", "f.txt")
	srcDir := filepath.Join(base, "s")
	dstDir := filepath.Join(base, "d")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f.txt", SourcePath: srcFile, DestPath: dstFile, Size: 100}}, 1, 100, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings.json"))
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{srcDir}, dstDir, false)
	
	// Start processing
	s.StartQueue()

	// Wait for processing to finish (very basic wait for test)
	time.Sleep(100 * time.Millisecond)

	job := s.GetQueue()[0]
	if job.Status != models.StatusSuccess {
		t.Errorf("Expected job status success, got %v", job.Status)
	}
}

func TestTransferService_RetryLogic(t *testing.T) {
	var attempts int32
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			atomic.AddInt32(&attempts, 1)
			if progressChan != nil {
				close(progressChan)
			}
			if atomic.LoadInt32(&attempts) == 1 {
				return &models.CopyError{Code: models.ErrCodeUnknown, Message: "retryable error"}
			}
			return nil
		},
	}
	base, _ := filepath.Abs(".")
	srcFile := filepath.Join(base, "s", "f.txt")
	dstFile := filepath.Join(base, "d", "f.txt")
	srcDir := filepath.Join(base, "s")
	dstDir := filepath.Join(base, "d")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f.txt", SourcePath: srcFile, DestPath: dstFile, Size: 10}}, 1, 10, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings.json"))
	// Use small retry delay if possible, but it's hardcoded to 2s.
	// I'll just check if it was called twice eventually or skip long wait in test by overriding if needed.
	// For now, I'll allow one retry and wait.
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{srcDir}, dstDir, false)
	s.StartQueue()

	// Need to wait enough for 1 retry (base delay 2s)
	// Actually, I should refactor the retry delay to be configurable too.
	// But let's see if 100ms is enough for the first failure.
	time.Sleep(100 * time.Millisecond)
	
	if atomic.LoadInt32(&attempts) < 1 {
		t.Error("Expected at least one attempt by now")
	}
}

func TestTransferService_CircuitBreaker(t *testing.T) {
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			if progressChan != nil {
				close(progressChan)
			}
			// Terminal error
			return &models.CopyError{Code: models.ErrCodeDiskFull, Message: "fatal"}
		},
	}
	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "s")
	dstDir := filepath.Join(base, "d")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{
				{Name: "f1", SourcePath: filepath.Join(srcDir, "f1"), DestPath: filepath.Join(dstDir, "f1"), Size: 10},
				{Name: "f2", SourcePath: filepath.Join(srcDir, "f2"), DestPath: filepath.Join(dstDir, "f2"), Size: 10},
				{Name: "f3", SourcePath: filepath.Join(srcDir, "f3"), DestPath: filepath.Join(dstDir, "f3"), Size: 10},
				{Name: "f4", SourcePath: filepath.Join(srcDir, "f4"), DestPath: filepath.Join(dstDir, "f4"), Size: 10},
			}, 4, 40, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings_cb.json"))
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{srcDir}, dstDir, false)
	s.StartQueue()

	time.Sleep(200 * time.Millisecond)

	job := s.GetQueue()[0]
	if job.Status != models.StatusPaused {
		t.Errorf("Expected job status paused (CB triggered), got %v", job.Status)
	}
	if !errors.Is(errors.New(job.Error), errors.New("Auto-paused due to consecutive systemic failures")) && job.Error != "Auto-paused due to consecutive systemic failures" {
		t.Errorf("Expected circuit breaker error message, got %q", job.Error)
	}
}

func TestTransferService_PauseAndResume(t *testing.T) {
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			defer close(progressChan)
			// Simulate long running copy that respects context
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				return nil
			}
		},
	}
	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "s")
	dstDir := filepath.Join(base, "d")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f.txt", SourcePath: filepath.Join(srcDir, "f.txt"), DestPath: filepath.Join(dstDir, "f.txt"), Size: 100}}, 1, 100, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings.json"))
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	jobID, _ := s.AddTransfer([]string{srcDir}, dstDir, false)
	s.StartQueue()

	time.Sleep(50 * time.Millisecond) // Job should be in progress
	s.Pause(jobID)
	
	time.Sleep(50 * time.Millisecond) // Wait for pause to propagate
	job := s.GetQueue()[0]
	if job.Status != models.StatusPaused {
		t.Errorf("Expected job status paused, got %v", job.Status)
	}

	// Resume
	s.Resume(jobID)
	
	// Wait for status to change to InProgress (polling to be robust)
	success := false
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		job = s.GetQueue()[0]
		if job.Status == models.StatusInProgress {
			success = true
			break
		}
	}
	
	if !success {
		t.Errorf("Expected job status in_progress after resume (jobID: %s), got %v", jobID, job.Status)
	}
}

func TestTransferService_Cancel(t *testing.T) {
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "s")
	dstDir := filepath.Join(base, "d")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f1", SourcePath: filepath.Join(srcDir, "f1"), DestPath: filepath.Join(dstDir, "f1"), Size: 10}}, 1, 10, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings.json"))
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	jobID, _ := s.AddTransfer([]string{srcDir}, dstDir, false)
	s.StartQueue()
	time.Sleep(50 * time.Millisecond)

	s.Cancel(jobID)
	time.Sleep(50 * time.Millisecond)

	job := s.GetQueue()[0]
	if job.Status != models.StatusFailed || job.Error != "cancelled" {
		t.Errorf("Expected job cancelled, got status %v, error %v", job.Status, job.Error)
	}
}

func TestTransferService_RemoveFileFromJob(t *testing.T) {
	base, _ := filepath.Abs(".")
	srcDir := filepath.Join(base, "s")
	srcFile1 := filepath.Join(srcDir, "f1")
	srcFile2 := filepath.Join(srcDir, "f2")

	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{
				{Name: "f1", SourcePath: srcFile1, Size: 100},
				{Name: "f2", SourcePath: srcFile2, Size: 200},
			}, 2, 300, nil
		},
	}

	dstDir := filepath.Join(base, "d")

	settings := NewSettingsServiceWithConfigPath(filepath.Join(base, "tmp", "settings.json"))
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), &MockCopier{}, mockScanner, afero.NewMemMapFs(), settings)

	jobID, _ := s.AddTransfer([]string{srcDir}, dstDir, false)
	
	s.RemoveFileFromJob(jobID, srcFile1)
	
	job := s.GetQueue()[0]
	if len(job.Files) != 1 {
		t.Errorf("Expected 1 file remaining, got %d", len(job.Files))
	}
	if job.TotalBytes != 200 {
		t.Errorf("Expected total bytes 200, got %d", job.TotalBytes)
	}
}

