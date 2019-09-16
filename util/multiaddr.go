package util

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
)

// GetMultiAddrsFromBytes take  [][]byte listeners and convert them into []Multiaddr format
func GetMultiAddrsFromBytes(listners [][]byte) []ma.Multiaddr {
	var maAddrs []ma.Multiaddr
	for _, addr := range listners {
		maAddr, err := ma.NewMultiaddrBytes(addr)
		if nil == err {
			maAddrs = append(maAddrs, maAddr)
		}
	}
	return maAddrs
}

// GetBytesFromMultiaddr take []Multiaddr format listeners and convert them into   [][]byte
func GetBytesFromMultiaddr(listners []ma.Multiaddr) [][]byte {
	var byteAddrs [][]byte
	for _, addr := range listners {
		byteAddrs = append(byteAddrs, addr.Bytes())
	}
	return byteAddrs
}

// ByteAddrsToString take an [][]byte and convert them it to multiAddress and return its presented string
func ByteAddrsToString(addrs [][]byte) []string {
	var addrsStr []string
	for _, addr := range addrs {
		newAddr, err := ma.NewMultiaddrBytes(addr)
		if nil == err {
			addrsStr = append(addrsStr, newAddr.String())
		}
	}
	return addrsStr
}

// PrintMaAddrs print out all ma with a new line seperater
func PrintMaAddrs(addrs []ma.Multiaddr) string {
	var stringAddr string
	for _, addr := range addrs {
		stringAddr = fmt.Sprintf("%s%s\n", stringAddr, addr.String())
	}
	return stringAddr
}
