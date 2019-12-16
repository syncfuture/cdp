package cdp

import (
	"net"
	"strconv"
	"time"
)

func isPortOpen(port int) bool {
	_, err := net.DialTimeout("tcp", "localhost:"+strconv.Itoa(port), time.Millisecond*500)
	if err != nil {
		return false
	}
	return true
}
