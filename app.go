package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/jdrews/certicopy/internal/services"
	"github.com/spf13/viper"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	transferService *services.TransferService
	settingsService *services.SettingsService
}

// NewApp creates a new App application struct
func NewApp() *App {
	settings := services.NewSettingsService()
	return &App{
		transferService: services.NewTransferService(settings),
		settingsService: settings,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.transferService.SetContext(ctx)

	// Process CLI transfers if any
	a.processCLITransfers()
}

func (a *App) processCLITransfers() {
	transfers := viper.GetStringSlice("transfer")
	overwrite := viper.GetBool("overwrite")
	hashAlgo := viper.GetString("hash")
	bufferSizeKB := viper.GetInt("buffer")

	// Apply CLI overrides to settings if provided
	if hashAlgo != "" || bufferSizeKB > 0 {
		settings := a.settingsService.Get()
		if hashAlgo != "" {
			settings.HashAlgorithm = hashAlgo
		}
		if bufferSizeKB > 0 {
			settings.BufferSize = bufferSizeKB * 1024
		}
		a.settingsService.Save(settings)
	}

	for _, t := range transfers {
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			fmt.Printf("Invalid transfer format: %s. Expected src:dst\n", t)
			continue
		}
		src, dst := parts[0], parts[1]
		_, err := a.transferService.AddTransfer([]string{src}, dst, overwrite)
		if err != nil {
			fmt.Printf("Failed to add CLI transfer %s -> %s: %v\n", src, dst, err)
		}
	}

	if len(transfers) > 0 {
		a.transferService.StartQueue()
	}
}

// AddTransferToQueue adds a transfer to the queue
func (a *App) AddTransferToQueue(sources []string, dest string, overwrite bool) (string, error) {
	return a.transferService.AddTransfer(sources, dest, overwrite)
}

// StartQueue starts the transfer queue processing
func (a *App) StartQueue() {
	a.transferService.StartQueue()
}

// GetQueue returns the current transfer queue
func (a *App) GetQueue() []*models.TransferJob {
	return a.transferService.GetQueue()
}

// PauseTransfer pauses the specified transfer
func (a *App) PauseTransfer(jobID string) {
	a.transferService.Pause(jobID)
}

// ResumeTransfer resumes the specified transfer
func (a *App) ResumeTransfer(jobID string) {
	a.transferService.Resume(jobID)
}

// CancelTransfer cancels the specified transfer
func (a *App) CancelTransfer(jobID string) {
	a.transferService.Cancel(jobID)
}

// SelectSource opens a dialog to select a source directory
func (a *App) SelectSource() ([]string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Source Directory",
	})
	if err != nil {
		fmt.Printf("SelectSource error: %v\n", err)
		return nil, err
	}
	if selection == "" {
		fmt.Println("SelectSource cancelled")
		return nil, nil // API expects empty/nil if cancelled
	}
	fmt.Printf("SelectSource selected: %s\n", selection)
	return []string{selection}, nil
}

// SelectDestination opens a dialog to select destination directory
func (a *App) SelectDestination() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Destination Directory",
	})
	if err != nil {
		fmt.Printf("SelectDestination error: %v\n", err)
		return "", err
	}
	if selection == "" {
		fmt.Println("SelectDestination cancelled")
		return "", nil // API expects empty string if cancelled
	}
	fmt.Printf("SelectDestination selected: %s\n", selection)
	return selection, nil
}

// GetSettings returns the current application settings
func (a *App) GetSettings() *models.Settings {
	return a.settingsService.Get()
}

// SaveSettings saves the application settings
func (a *App) SaveSettings(settings *models.Settings) error {
	return a.settingsService.Save(settings)
}
