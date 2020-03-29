// +build !windows

package fetcher

import "strings"

func isDisconnectedError(err error) bool {
	if err != nil {
		return false
	}

	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}

	return false
}
