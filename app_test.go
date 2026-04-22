package main

import (
	"path/filepath"
	"testing"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/jdrews/certicopy/internal/services"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func TestApp_AddTransferToQueue(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	// Setup test files
	srcFile := filepath.Join(base, "src", "file1.txt")
	dstDir := filepath.Join(base, "dst")
	_ = afero.WriteFile(fs, srcFile, []byte("hello"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, err := app.AddTransferToQueue([]string{srcFile}, dstDir, false)
	if err != nil {
		t.Fatalf("Failed to add transfer: %v", err)
	}

	if jobID == "" {
		t.Error("Expected non-empty jobID")
	}

	queueItems := app.GetQueue()
	if len(queueItems) != 1 {
		t.Errorf("Expected 1 queue item, got %d", len(queueItems))
	}
}

func TestApp_TransferActions(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	srcFile := filepath.Join(base, "src", "file1.txt")
	dstDir := filepath.Join(base, "dst")
	_ = afero.WriteFile(fs, srcFile, []byte("hello"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, _ := app.AddTransferToQueue([]string{srcFile}, dstDir, false)

	// Test Pause
	app.PauseTransfer(jobID)
	job := app.GetQueue()[0]
	if job.Status != models.StatusPaused {
		t.Errorf("Expected status paused, got %v", job.Status)
	}

	// Test Resume
	app.ResumeTransfer(jobID)
	job = app.GetQueue()[0]
	if job.Status != models.StatusPending && job.Status != models.StatusInProgress {
		t.Errorf("Expected status pending or in_progress, got %v", job.Status)
	}

	// Test Cancel
	app.CancelTransfer(jobID)
	job = app.GetQueue()[0]
	if job.Status != models.StatusFailed {
		t.Errorf("Expected status failed (cancelled), got %v", job.Status)
	}
}

func TestApp_Settings(t *testing.T) {
	fs := afero.NewMemMapFs()
	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")

	app := &App{
		settingsService: settings,
	}

	s := app.GetSettings()
	if s == nil {
		t.Fatal("Expected settings, got nil")
	}

	s.HashAlgorithm = "sha512"
	err := app.SaveSettings(s)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	s2 := app.GetSettings()
	if s2.HashAlgorithm != "sha512" {
		t.Errorf("Expected sha512, got %v", s2.HashAlgorithm)
	}
}

func TestApp_RemoveFileFromJob(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	srcFile1 := filepath.Join(base, "src", "file1.txt")
	srcFile2 := filepath.Join(base, "src", "file2.txt")
	dstDir := filepath.Join(base, "dst")
	_ = afero.WriteFile(fs, srcFile1, []byte("hello"), 0644)
	_ = afero.WriteFile(fs, srcFile2, []byte("world"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, _ := app.AddTransferToQueue([]string{srcFile1, srcFile2}, dstDir, false)
	
	app.RemoveFileFromJob(jobID, srcFile1)
	
	job := app.GetQueue()[0]
	if len(job.Files) != 1 {
		t.Errorf("Expected 1 file remaining, got %d", len(job.Files))
	}
	if job.Files[0].SourcePath != srcFile2 {
		t.Errorf("Expected %s to remain, got %s", srcFile2, job.Files[0].SourcePath)
	}
}

func TestApp_ProcessCLITransfers(t *testing.T) {
	fs := afero.NewMemMapFs()
	base, _ := filepath.Abs(".")
	srcFile := filepath.Join(base, "src", "file1.txt")
	dstFile := filepath.Join(base, "dst", "file1.txt")
	_ = afero.WriteFile(fs, srcFile, []byte("hello"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	// Use viper to simulate CLI flags
	viper.Set("transfer", []string{srcFile + "," + dstFile})
	
	app.processCLITransfers()
	
	queueItems := app.GetQueue()
	if len(queueItems) != 1 {
		t.Errorf("Expected 1 job from CLI, got %d", len(queueItems))
	}
	
	job := queueItems[0]
	if job.Destination != dstFile {
		t.Errorf("Expected destination %s, got %s", dstFile, job.Destination)
	}
}
