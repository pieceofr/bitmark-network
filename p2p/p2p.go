package p2p

import (
	"bitmark-network/messagebus"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
)

// global data
var globalData Node

// const
const (
	domainLocal   = "nodes.rachael.bitmark"
	domainBitamrk = "nodes.test.bitmark.com"
	domainTest    = "nodes.test.bitmark.com"
)

// StaticConnection - hardwired connections
// this is read from the configuration file
type StaticConnection struct {
	PublicKey string `gluamapper:"public_key" json:"public_key"`
	Address   string `gluamapper:"address" json:"address"`
}

// Configuration - a block of configuration data
// this is read from the configuration file
type Configuration struct {
	PublicIP           []string           `gluamapper:"publicip" json:"publicip"`
	NodeType           string             `gluamapper:"nodetype" json:"nodetype"`
	Port               int                `gluamapper:"port" json:"port"`
	DynamicConnections bool               `gluamapper:"dynamic_connections" json:"dynamic_connections"`
	PreferIPv6         bool               `gluamapper:"prefer_ipv6" json:"prefer_ipv6"`
	Listen             []string           `gluamapper:"listen" json:"listen"`
	Announce           []string           `gluamapper:"announce" json:"announce"`
	PrivateKey         string             `gluamapper:"private_key" json:"private_key"`
	PublicKey          string             `gluamapper:"public_key" json:"public_key"`
	Connect            []StaticConnection `gluamapper:"connect" json:"connect,omitempty"`
}

// Initialise initialize p2p module
func Initialise(configuration *Configuration) error {
	globalData.Lock()
	defer globalData.Unlock()
	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}
	globalData.log = logger.New("network")
	globalData.log.Info("starting…")
	// Create A p2p Node
	globalData.Setup(configuration)
	// Node setup
	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		&globalData,
	}
	globalData.background = background.Start(processes, globalData.log)
	return nil
}

// Run  wait for incoming requests, process them and reply
func (n *Node) Run(args interface{}, shutdown <-chan struct{}) {
	log := n.log
	log.Info("starting…")
	queue := messagebus.Bus.P2P.Chan()
	delay := time.After(nodeInitial)
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("Node Module received control item")
			switch item.Command {
			case "peer":
				messageOut := P2PMessage{Command: item.Command, Parameters: item.Parameters}
				msgBytes, err := proto.Marshal(&messageOut)
				err = n.MuticastStream.Publish(multicastingTopic, msgBytes)
				if err != nil {
					log.Errorf("multicast error: %v\n", err)
					break
				}
				log.Infof("<<--- multicasting PEER : %v\n", string(messageOut.Parameters[0]))
			}
		case <-delay:
			delay = time.After(nodeInterval)
			log.Infof("Node Module Interval")
		}
	}
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()
	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
