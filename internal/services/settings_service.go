package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/jdrews/certicopy/internal/core"
	"github.com/jdrews/certicopy/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// SettingsService handles loading and saving application settings
type SettingsService struct {
	settings   *models.Settings
	mutex      sync.RWMutex
	configPath string
	fs         afero.Fs
}

// NewSettingsService creates a new SettingsService
func NewSettingsService() *SettingsService {
	// Use OS filesystem for actual operations
	fs := afero.NewOsFs()

	// Determine config path
	configDir, err := os.UserConfigDir()
	configPath := "settings.json" // Default to current dir if config dir fails
	if err == nil {
		configDir = filepath.Join(configDir, "certicopy")
		// Ensure directory exists
		_ = fs.MkdirAll(configDir, 0755)
		configPath = filepath.Join(configDir, "settings.json")
	}

	s := &SettingsService{
		configPath: configPath,
		settings:   models.DefaultSettings(),
		fs:         fs,
	}
	// Try to load existing settings
	_ = s.Load()
	return s
}

// NewSettingsServiceWithConfigPath creates a new SettingsService with a specific config path
func NewSettingsServiceWithConfigPath(configPath string) *SettingsService {
	return &SettingsService{
		configPath: configPath,
		settings:   models.DefaultSettings(),
		fs:         afero.NewOsFs(),
	}
}

// NewSettingsServiceWithFs creates a new SettingsService with a specific filesystem for testing
func NewSettingsServiceWithFs(fs afero.Fs, configPath string) *SettingsService {
	return &SettingsService{
		configPath: configPath,
		settings:   models.DefaultSettings(),
		fs:         fs,
	}
}

// Load reads settings from disk
func (s *SettingsService) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := afero.ReadFile(s.fs, s.configPath)
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

	return afero.WriteFile(s.fs, s.configPath, data, 0644)
}

// Get returns the current settings
func (s *SettingsService) Get() *models.Settings {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.settings
}
