package proxy

import "strings"

const netErrUseOfCloseNetworkConnection = "use of closed network connection"

func isNetErrUseOfCloseNetworkConnection(err error) bool {
	if strings.Contains(err.Error(), netErrUseOfCloseNetworkConnection) {
		return true
	}
	return false
}
