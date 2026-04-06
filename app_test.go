package main

import (
	"testing"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/jdrews/certicopy/internal/services"
	"github.com/spf13/afero"
)

func TestApp_AddTransferToQueue(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Setup test files
	_ = afero.WriteFile(fs, "/src/file1.txt", []byte("hello"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, err := app.AddTransferToQueue([]string{"/src/file1.txt"}, "/dst", false)
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
	_ = afero.WriteFile(fs, "/src/file1.txt", []byte("hello"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, _ := app.AddTransferToQueue([]string{"/src/file1.txt"}, "/dst", false)

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
	_ = afero.WriteFile(fs, "/src/file1.txt", []byte("hello"), 0644)
	_ = afero.WriteFile(fs, "/src/file2.txt", []byte("world"), 0644)

	settings := services.NewSettingsServiceWithFs(fs, "/test/settings.json")
	queue := core.NewTransferQueue()
	copier := core.NewCopier(fs, fs)
	scanner := core.NewScanner(fs)
	ts := services.NewTransferServiceWithDeps(queue, copier, scanner, fs, settings)

	app := &App{
		transferService: ts,
		settingsService: settings,
	}

	jobID, _ := app.AddTransferToQueue([]string{"/src/file1.txt", "/src/file2.txt"}, "/dst", false)
	
	app.RemoveFileFromJob(jobID, "/src/file1.txt")
	
	job := app.GetQueue()[0]
	if len(job.Files) != 1 {
		t.Errorf("Expected 1 file remaining, got %d", len(job.Files))
	}
	if job.Files[0].SourcePath != "/src/file2.txt" {
		t.Errorf("Expected /src/file2.txt to remain, got %s", job.Files[0].SourcePath)
	}
}
