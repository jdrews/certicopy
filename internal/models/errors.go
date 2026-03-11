package models

import "fmt"

// ErrorCode represents a specific type of error
type ErrorCode string

const (
	ErrCodeNetworkDisconnect ErrorCode = "NETWORK_DISCONNECT"
	ErrCodePermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrCodeDiskFull          ErrorCode = "DISK_FULL"
	ErrCodeChecksumMismatch  ErrorCode = "CHECKSUM_MISMATCH"
	ErrCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrCodeUnknown           ErrorCode = "UNKNOWN_ERROR"
)

// CopyError is a structured error containing context for the UI/Service layers
type CopyError struct {
	Code    ErrorCode
	Message string
	Err     error
	Path    string
}

func (e *CopyError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *CopyError) Unwrap() error {
	return e.Err
}

// IsAutoRetryable determines if this error should be automatically retried with exponential backoff
func (e *CopyError) IsAutoRetryable() bool {
	// We only auto-retry network disconnects or unknown transient errors.
	// We do NOT auto-retry Permission Denied, Disk Full, or Not Found as these are terminal and require user intervention
	switch e.Code {
	case ErrCodeNetworkDisconnect:
		return true
	default:
		return false
	}
}
