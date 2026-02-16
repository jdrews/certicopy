package models

import "time"

// TransferJob represents a batch of files to be transferred
type TransferJob struct {
	ID          string         `json:"id"`
	Sources     []string       `json:"sources"`
	Destination string         `json:"destination"`
	Status      TransferStatus `json:"status"`
	TotalFiles  int64          `json:"totalFiles"`
	TotalBytes  int64          `json:"totalBytes"`
	BytesCopied int64          `json:"bytesCopied"`
	Files       []*FileInfo    `json:"files"`
	CreatedAt   time.Time      `json:"createdAt"`
	StartedAt   time.Time      `json:"startedAt"`
	CompletedAt time.Time      `json:"completedAt"`
	Error       string         `json:"error,omitempty"`
}

// Stats returns the current statistics for the job
func (j *TransferJob) Stats() (filesCopied int64, bytesCopied int64, failed int64) {
	for _, f := range j.Files {
		if f.Status == StatusSuccess {
			filesCopied++
			bytesCopied += f.Size
		} else if f.Status == StatusFailed {
			failed++
		}
	}
	return
}
