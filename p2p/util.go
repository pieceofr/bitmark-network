package p2p

import (
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	taggedPublic  = "PUBLIC:"
	taggedPrivate = "PRIVATE:"
	publicLength  = 32
	privateLength = 64
)

// ParseHostPort - parse host:port
func ParseHostPort(hostPort string) (string, string, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if nil != err {
		return "", "", err
	}
	IP := strings.Trim(host, " ")
	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", "", err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", "", fault.ErrInvalidPortNumber
	}
	return IP, strconv.Itoa(numericPort), nil
}

// addrToConnAddr remove protocol ID and node ID
func addrToConnAddr(addr string) string {
	addrSlice := strings.Split(addr, "/")
	var retAddr string
	if len(addrSlice) > 4 {
		for idx, addr := range addrSlice[:len(addrSlice)-2] {
			if idx == (len(addrSlice) - 3) {
				retAddr = retAddr + addr
			} else {
				retAddr = retAddr + addr + "/"
			}
		}
	}
	return retAddr
}

func shortID(id string) string {
	if len(id) > 11 {
		return id[len(id)-11 : len(id)-1]
	}
	return id
}
