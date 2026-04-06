package models

import (
	"testing"
)

func TestTransferJob_Stats(t *testing.T) {
	job := &TransferJob{
		Files: []*FileInfo{
			{Name: "file1", Size: 100, Status: StatusSuccess},
			{Name: "file2", Size: 200, Status: StatusSuccess},
			{Name: "file3", Size: 300, Status: StatusFailed},
			{Name: "file4", Size: 400, Status: StatusPending},
			{Name: "file5", Size: 500, Status: StatusInProgress},
		},
	}

	filesCopied, bytesCopied, failed := job.Stats()

	if filesCopied != 2 {
		t.Errorf("Expected 2 files copied, got %d", filesCopied)
	}
	if bytesCopied != 300 {
		t.Errorf("Expected 300 bytes copied, got %d", bytesCopied)
	}
	if failed != 1 {
		t.Errorf("Expected 1 failed file, got %d", failed)
	}
}

func TestTransferJob_Stats_Empty(t *testing.T) {
	job := &TransferJob{
		Files: []*FileInfo{},
	}

	filesCopied, bytesCopied, failed := job.Stats()

	if filesCopied != 0 || bytesCopied != 0 || failed != 0 {
		t.Errorf("Expected all zeros for empty job, got %d, %d, %d", filesCopied, bytesCopied, failed)
	}
}
