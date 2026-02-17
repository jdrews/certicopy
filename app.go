package main

import (
	"context"
	"fmt"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/jdrews/certicopy/internal/services"
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
	return &App{
		transferService: services.NewTransferService(),
		settingsService: services.NewSettingsService(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.transferService.SetContext(ctx)
}

// AddTransferToQueue adds a transfer to the queue
func (a *App) AddTransferToQueue(sources []string, dest string) (string, error) {
	return a.transferService.AddTransfer(sources, dest)
}

// StartQueue starts the transfer queue processing
func (a *App) StartQueue() {
	a.transferService.StartQueue()
}

// GetQueue returns the current transfer queue
func (a *App) GetQueue() []*models.TransferJob {
	return a.transferService.GetQueue()
}

// PauseTransfer pauses the current transfer
func (a *App) PauseTransfer() {
	// TODO: Implement pause logic
	a.transferService.Pause()
}

// ResumeTransfer resumes the current transfer
func (a *App) ResumeTransfer() {
	// TODO: Implement resume logic
	a.transferService.Resume()
}

// CancelTransfer cancels the current transfer
func (a *App) CancelTransfer() {
	a.transferService.Cancel()
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
