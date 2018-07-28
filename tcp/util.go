package tcp

import (
	"strings"
)

func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}
	// TODO: remove this string search and be more like the Windows
	// case below. That might involve modifying the standard library
	// to return better error types.
	str := err.Error()
	if strings.Contains(str, "use of closed network connection") {
		return true
	}
	return false
}

