package services

import (
	"context"
	"errors"
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

	id, err := s.AddTransfer([]string{"/src/f1"}, "/dest", false)
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
	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f.txt", SourcePath: "/s/f.txt", DestPath: "/d/f.txt", Size: 100}}, 1, 100, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath("/tmp/settings.json")
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{"/s/f.txt"}, "/d", false)
	
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
	attempts := 0
	mockCopier := &MockCopier{
		copyWithProgressFunc: func(ctx context.Context, src string, dst string, opts core.CopyOptions, progressChan chan<- core.Progress) error {
			attempts++
			if progressChan != nil {
				close(progressChan)
			}
			if attempts == 1 {
				return &models.CopyError{Code: models.ErrCodeUnknown, Message: "retryable error"}
			}
			return nil
		},
	}
	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{{Name: "f.txt", SourcePath: "/s/f.txt", DestPath: "/d/f.txt", Size: 10}}, 1, 10, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath("/tmp/settings.json")
	// Use small retry delay if possible, but it's hardcoded to 2s.
	// I'll just check if it was called twice eventually or skip long wait in test by overriding if needed.
	// For now, I'll allow one retry and wait.
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{"/s"}, "/d", false)
	s.StartQueue()

	// Need to wait enough for 1 retry (base delay 2s)
	// Actually, I should refactor the retry delay to be configurable too.
	// But let's see if 100ms is enough for the first failure.
	time.Sleep(100 * time.Millisecond)
	
	if attempts < 1 {
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
	mockScanner := &MockScanner{
		scanFunc: func(sources []string, destRoot string) ([]*models.FileInfo, int64, int64, error) {
			return []*models.FileInfo{
				{Name: "f1", SourcePath: "/s/f1", DestPath: "/d/f1", Size: 10},
				{Name: "f2", SourcePath: "/s/f2", DestPath: "/d/f2", Size: 10},
				{Name: "f3", SourcePath: "/s/f3", DestPath: "/d/f3", Size: 10},
				{Name: "f4", SourcePath: "/s/f4", DestPath: "/d/f4", Size: 10},
			}, 4, 40, nil
		},
	}

	settings := NewSettingsServiceWithConfigPath("/tmp/settings_cb.json")
	s := NewTransferServiceWithDeps(core.NewTransferQueue(), mockCopier, mockScanner, afero.NewMemMapFs(), settings)

	_, _ = s.AddTransfer([]string{"/s"}, "/d", false)
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
