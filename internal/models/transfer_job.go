package models

// TransferJob represents a batch of files to be transferred
type TransferJob struct {
	ID          string         `json:"id"`
	Sources     []string       `json:"sources"`
	Destination string         `json:"destination"`
	Overwrite   bool           `json:"overwrite"`
	Status      TransferStatus `json:"status"`
	TotalFiles  int64          `json:"totalFiles"`
	TotalBytes  int64          `json:"totalBytes"`
	BytesCopied int64          `json:"bytesCopied"`
	Files       []*FileInfo    `json:"files"`
	CreatedAt   int64          `json:"createdAt"`   // Unix milliseconds
	StartedAt   int64          `json:"startedAt"`   // Unix milliseconds
	CompletedAt int64          `json:"completedAt"` // Unix milliseconds
	Error       string         `json:"error,omitempty"`
	ErrorCode   string         `json:"errorCode,omitempty"`
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
