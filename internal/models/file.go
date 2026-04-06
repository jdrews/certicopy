package models

// TransferStatus represents the current state of a file transfer
type TransferStatus string

const (
	StatusPending    TransferStatus = "pending"
	StatusInProgress TransferStatus = "in_progress"
	StatusSuccess    TransferStatus = "success"
	StatusFailed     TransferStatus = "failed"
	StatusSkipped    TransferStatus = "skipped"
	StatusPaused     TransferStatus = "paused"
	StatusHashing    TransferStatus = "hashing"
)

// FileInfo contains metadata and transfer status for a single file
type FileInfo struct {
	JobID        string         `json:"jobId"`
	SourcePath   string         `json:"sourcePath"`
	DestPath     string         `json:"destPath"`
	Name         string         `json:"name"`
	Size         int64          `json:"size"`
	ModTime      int64          `json:"modTime"` // Unix milliseconds
	Status       TransferStatus `json:"status"`
	SourceHash   string         `json:"sourceHash"`
	DestHash     string         `json:"destHash"`
	ErrorMessage      string         `json:"errorMessage,omitempty"`
	ErrorCode         string         `json:"errorCode,omitempty"`
	BytesCopied       int64          `json:"bytesCopied"`
	TransferCompleted bool           `json:"transferCompleted"`
	EndHashVerified   bool           `json:"endHashVerified"`
}
