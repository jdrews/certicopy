package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/sirupsen/logrus"
)

// SettingsService handles loading and saving application settings
type SettingsService struct {
	settings   *models.Settings
	mutex      sync.RWMutex
	configPath string
}

// NewSettingsService creates a new SettingsService
func NewSettingsService() *SettingsService {
	// Determine config path
	home, err := os.UserHomeDir()
	configPath := "settings.json" // Default to current dir if home fails
	if err == nil {
		configDir := filepath.Join(home, ".config", "certicopy")
		// Ensure directory exists
		_ = os.MkdirAll(configDir, 0755)
		configPath = filepath.Join(configDir, "settings.json")
	}

	s := &SettingsService{
		configPath: configPath,
		settings:   models.DefaultSettings(),
	}
	// Try to load existing settings
	_ = s.Load()
	return s
}

// Load reads settings from disk
func (s *SettingsService) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		// If file doesn't exist, we just stick with defaults
		core.Log.WithField("path", s.configPath).Info("No settings file found, using defaults")
		return err
	}

	if err := json.Unmarshal(data, s.settings); err != nil {
		return err
	}
	core.Log.WithField("path", s.configPath).Info("Settings loaded from disk")
	return nil
}

// Save writes settings to disk
func (s *SettingsService) Save(settings *models.Settings) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.settings = settings
	core.Log.WithFields(logrus.Fields{
		"hashAlgo":   settings.HashAlgorithm,
		"bufferSize": settings.BufferSize,
		"overwrite":  settings.Overwrite,
	}).Info("Settings saved")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

// Get returns the current settings
func (s *SettingsService) Get() *models.Settings {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.settings
}
