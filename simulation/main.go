package main

import (
	"bitmark-network/p2p"
	"bitmark-network/routing"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/logger"
)

// GlobalMockConfiguration Markup global configuration
type GlobalMockConfiguration struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	Peering       p2p.Configuration    `gluamapper:"peering" json:"peering"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
	PidFile       string               `gluamapper:"pidfile" json:"pidfile"`
	Chain         string               `gluamapper:"chain" json:"chain"`
	Nodes         string               `gluamapper:"nodes" json:"nodes"`
}

func main() {
	var globalConf GlobalMockConfiguration
	path := filepath.Join(os.Getenv("PWD"), "p2p.conf")
	err := configuration.ParseConfigurationFile(path, &globalConf)
	if err != nil {
		fmt.Println("Error:", err)
	}
	// start logging
	if err = logger.Initialise(globalConf.Logging); nil != err {
		panic(err)
	}
	defer logger.Finalise()
	err = routing.Initialise(getDomainName(globalConf), getPeerFile(globalConf.Chain))
	if nil != err {
		panic(fmt.Sprintf("peer initialise error: %s", err))
	}
	defer routing.Finalise()
	err = p2p.Initialise(&globalConf.Peering)
	if nil != err {
		panic(fmt.Sprintf("peer initialise error: %s", err))
	}
	defer p2p.Finalise()
}

func getPeerFile(chain string) string {
	curPath := os.Getenv("PWD")
	switch chain {
	case "bitmark":
		return path.Join(curPath, "peers-bitmark")
	case "testing":
		return path.Join(curPath, "peers-testing")
	case "local":
		return path.Join(curPath, "peers-local")
	default:
		panic("invalid chain name")
	}
}

func getDomainName(masterConfiguration GlobalMockConfiguration) string {
	nodesDomain := ""
	switch masterConfiguration.Nodes {
	case "":
		panic("nodes cannot be blank choose from: none, chain or sub.domain.tld")
	case "none":
		panic("nodes cannot be blank choose from: none, chain or sub.domain.tld")
	case "chain":
		switch cn := masterConfiguration.Chain; cn { // ***** FIX THIS: is there a better way?
		case "local":
			nodesDomain = masterConfiguration.Nodes
		case "testing":
			nodesDomain = "nodes.test.bitmark.com"
		case "bitmark":
			nodesDomain = "nodes.live.bitmark.com"
		default:
			panic(fmt.Sprintf("unexpected chain name: %q", cn))
		}
	default:
		// domain names are complex to validate so just rely on
		// trying to fetch the TXT records for validation
		nodesDomain = masterConfiguration.Nodes // just assume it is a domain name
	}
	fmt.Println(nodesDomain)
	return nodesDomain
}
