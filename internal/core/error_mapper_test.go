package core

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/jdrews/certicopy/internal/models"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		path     string
		expected models.ErrorCode
	}{
		{
			name:     "Nil Error",
			err:      nil,
			path:     "test.txt",
			expected: models.ErrCodeUnknown, // should not happen really
		},
		{
			name:     "Network Disconnect (ENOTCONN)",
			err:      &os.SyscallError{Err: syscall.ENOTCONN},
			path:     "//server/share/file.txt",
			expected: models.ErrCodeNetworkDisconnect,
		},
		{
			name:     "Network Connect Timeout (ETIMEDOUT)",
			err:      &net.OpError{Err: syscall.ETIMEDOUT},
			path:     "//server/share/file.txt",
			expected: models.ErrCodeNetworkDisconnect,
		},
		{
			name:     "Permission Denied (fs.ErrPermission)",
			err:      fs.ErrPermission,
			path:     "/root/secret.txt",
			expected: models.ErrCodePermissionDenied,
		},
		{
			name:     "Permission Denied (syscall.EACCES)",
			err:      &os.PathError{Err: syscall.EACCES},
			path:     "/readonly/file.txt",
			expected: models.ErrCodePermissionDenied,
		},
		{
			name:     "File Not Found (fs.ErrNotExist)",
			err:      fs.ErrNotExist,
			path:     "/tmp/missing.txt",
			expected: models.ErrCodeNotFound,
		},
		{
			name:     "Disk Full (ENOSPC)",
			err:      &os.PathError{Err: syscall.ENOSPC},
			path:     "/mnt/full/file.txt",
			expected: models.ErrCodeDiskFull,
		},
		{
			name:     "String match: no space left",
			err:      errors.New("write error: no space left on device"),
			path:     "/mnt/full/file.txt",
			expected: models.ErrCodeDiskFull,
		},
		{
			name:     "Unknown Error",
			err:      errors.New("some random error"),
			path:     "file.txt",
			expected: models.ErrCodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyErr := MapError(tt.err, "ReadFile", tt.path)
			if copyErr == nil && tt.err != nil {
				t.Fatalf("MapError returned nil for non-nil error")
			}
			if copyErr != nil && copyErr.Code != tt.expected {
				t.Errorf("expected error code %v, got %v", tt.expected, copyErr.Code)
			}
		})
	}
}
