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
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		transferService: services.NewTransferService(),
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
	// Note: We need to expose TransferJob type in a way Wails can generate TS bindings.
	// Since TransferJob is in internal/models, Wails might not pick it up if not imported?
	// Actually Wails v2 generates models automatically for returned types.
	// But we need to make sure the type is exported.
	return a.transferService.GetQueue()
}
