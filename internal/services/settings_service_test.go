package services

import (
	"encoding/json"
	"testing"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/spf13/afero"
)

func TestSettingsService_SaveAndLoad(t *testing.T) {
	fs := afero.NewMemMapFs()
	configPath := "/test/settings.json"
	s := NewSettingsServiceWithFs(fs, configPath)

	newSettings := models.DefaultSettings()
	newSettings.HashAlgorithm = "sha256"
	newSettings.BufferSize = 4096
	newSettings.Overwrite = true

	if err := s.Save(newSettings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Verify file was created and content is correct
	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var savedSettings models.Settings
	if err := json.Unmarshal(data, &savedSettings); err != nil {
		t.Fatalf("Failed to unmarshal saved settings: %v", err)
	}

	if savedSettings.HashAlgorithm != "sha256" || savedSettings.BufferSize != 4096 || !savedSettings.Overwrite {
		t.Errorf("Saved settings mismatch: %+v", savedSettings)
	}

	// Test loading
	s2 := NewSettingsServiceWithFs(fs, configPath)
	if err := s2.Load(); err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	loaded := s2.Get()
	if loaded.HashAlgorithm != "sha256" || loaded.BufferSize != 4096 || !loaded.Overwrite {
		t.Errorf("Loaded settings mismatch: %+v", loaded)
	}
}

func TestSettingsService_Load_Fail(t *testing.T) {
	fs := afero.NewMemMapFs()
	configPath := "/nonexistent/settings.json"
	s := NewSettingsServiceWithFs(fs, configPath)

	err := s.Load()
	if err == nil {
		t.Error("Expected error loading non-existent file, got nil")
	}
}
