package fetcher

import (
	"net"
	"os"
	"syscall"
)

func isDisconnectedError(err error) bool {
	if err == nil {
		return false
	}

	oe, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	se, ok := oe.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	errno, ok := se.Err.(syscall.Errno)
	if !ok {
		return false
	}
	if errno == syscall.WSAECONNABORTED || errno == syscall.WSAECONNRESET {
		return true
	}

	return false
}
