package main

import (
	"context"

	"github.com/jdrews/certicopy/internal/models"
	"github.com/jdrews/certicopy/internal/services"
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

// GetSettings returns the current application settings
func (a *App) GetSettings() *models.Settings {
	return a.settingsService.Get()
}

// SaveSettings saves the application settings
func (a *App) SaveSettings(settings *models.Settings) error {
	return a.settingsService.Save(settings)
}
