package models

import (
	"testing"
)

func TestCopyError_IsAutoRetryable(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want bool
	}{
		{ErrCodeNetworkDisconnect, true},
		{ErrCodeDiskFull, false},
		{ErrCodeChecksumMismatch, false},
		{ErrCodeUnknown, false},
		{ErrCodePermissionDenied, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := &CopyError{Code: tt.code}
			if got := e.IsAutoRetryable(); got != tt.want {
				t.Errorf("IsAutoRetryable() for %v = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}
