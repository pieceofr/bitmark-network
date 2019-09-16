package p2p

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
)

func announceAddr(announce []ma.Multiaddr, protocol string, id peer.ID) []ma.Multiaddr {
	maAddrs := []ma.Multiaddr{}
	p2pMa, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%s", protocol, id.String()))
	if err != nil {
		return nil
	}
	for _, addr := range announce {
		maAddrs = append(maAddrs, addr.Encapsulate(p2pMa))
	}
	return maAddrs
}

func ipPortAddrsToMaAddrs(addrs []string) []ma.Multiaddr {
	maListener := []ma.Multiaddr{}
	for _, addr := range addrs {
		v, ip, port, err := parseIPPort(addr)
		if err == nil {
			if "ipv4" == v {
				listenAddrIPV4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%v", ip, port))
				if err == nil {
					maListener = append(maListener, listenAddrIPV4)
				}
			} else {
				listenAddrIPV6, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%v", ip, port))
				if err == nil {
					maListener = append(maListener, listenAddrIPV6)
				}
			}
		}
	}
	return maListener
}

func parseIPPort(hostPort string) (v string, ip string, port uint16, err error) {
	host, portStr, err := net.SplitHostPort(hostPort)
	if nil != err {
		return "", "", 0, fault.ErrInvalidIpAddress
	}

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return "", "", 0, fault.ErrInvalidIpAddress
	}
	if nil != IP.To4() {
		v = "ipv4"
	} else {
		v = "ipv6"
	}

	numericPort, err := strconv.Atoi(strings.Trim(portStr, " "))
	if nil != err {
		return "", "", 0, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", "", 0, fault.ErrInvalidPortNumber
	}
	return v, IP.String(), uint16(numericPort), nil
}
