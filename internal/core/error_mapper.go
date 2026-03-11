package core

import (
	"errors"
	"io/fs"
	"strings"
	"syscall"

	"github.com/jdrews/certicopy/internal/models"
)

// MapError translates standard Go errors into our custom structured CopyError
func MapError(err error, path string, defaultMessage string) *models.CopyError {
	if err == nil {
		return nil
	}

	// Unpack base error if it's already a copy error
	var copyErr *models.CopyError
	if errors.As(err, &copyErr) {
		return copyErr
	}

	code := models.ErrCodeUnknown

	// Check for network disconnects (syscall/net types)
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		if sysErr == syscall.ENOTCONN || sysErr == syscall.EHOSTUNREACH || sysErr == syscall.ECONNRESET || sysErr == syscall.ECONNABORTED || sysErr == syscall.ETIMEDOUT {
			code = models.ErrCodeNetworkDisconnect
		} else if isWindowsNetworkError(sysErr) {
			code = models.ErrCodeNetworkDisconnect
		} else if sysErr == syscall.ENOSPC { // No space left on device
			code = models.ErrCodeDiskFull
		} else if sysErr == syscall.EACCES || sysErr == syscall.EPERM {
			code = models.ErrCodePermissionDenied
		}
	} else if errors.Is(err, fs.ErrPermission) {
		code = models.ErrCodePermissionDenied
	} else if errors.Is(err, fs.ErrNotExist) {
		code = models.ErrCodeNotFound
	} else {
		// Sometimes network errors get wrapped opaquely or are just strings matching
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "no space left") || strings.Contains(errMsg, "disk full") {
			code = models.ErrCodeDiskFull
		} else if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access denied") {
			code = models.ErrCodePermissionDenied
		} else if strings.Contains(errMsg, "no such file") {
			code = models.ErrCodeNotFound
		}
	}

	return &models.CopyError{
		Code:    code,
		Message: defaultMessage,
		Err:     err,
		Path:    path,
	}
}

func isWindowsNetworkError(sysErr syscall.Errno) bool {
	// 55 = ERROR_DEV_NOT_EXIST
	// 64 = ERROR_NETNAME_DELETED
	// 59 = ERROR_UNEXP_NET_ERR
	val := uintptr(sysErr)
	return val == 55 || val == 64 || val == 59
}
