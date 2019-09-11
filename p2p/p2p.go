package p2p

import (
	"bitmark-network/messagebus"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
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
	PublicKey          string             `gluamapper:"public_key" json:"public_key"` //TODO : REMOVE
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
	globalData.log = logger.New("p2p")
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
			log.Infof("-><- P2P recieve commend:%s", item.Command)
			switch item.Command {
			case "peer":
				messageOut := P2PMessage{Command: item.Command, Parameters: item.Parameters}
				msgBytes, err := proto.Marshal(&messageOut)
				if err != nil {
					log.Errorf("Marshal Message Error: %v\n", err)
					break
				}
				id, err := peer.IDFromBytes(messageOut.Parameters[0])
				if err != nil {
					log.Errorf("Inavalid ID format:%v", err)
					break
				}
				err = n.MuticastStream.Publish(multicastingTopic, msgBytes)
				if err != nil {
					log.Errorf("Multicast Publish Error: %v\n", err)
					break
				}
				log.Infof("<<--- multicasting PEER : %v\n", id.ShortString())

			default:
				if "N1" == item.Command || "N3" == item.Command || "X1" == item.Command || "X2" == item.Command ||
					"X3" == item.Command || "X4" == item.Command || "X5" == item.Command || "X6" == item.Command ||
					"X7" == item.Command || "P1" == item.Command || "P2" == item.Command {
					// save to node peer and connect ; 				messagebus.Bus.P2P.Send(names[i], peer.peerID, peer.listeners)
					pbListensers := Addrs{}
					proto.Unmarshal(item.Parameters[1], &pbListensers)
					for _, listener := range pbListensers.Address {
						if maListener, err := ma.NewMultiaddrBytes(listener); nil == err {
							//	n.log.Infof("command:%s request add peerid:%s", item.Command, string(item.Parameters[0]))
							n.addPeer(peer.ID(string(item.Parameters[0])), maListener)
						} else {
							log.Errorf("Announce Message Error:%v", err)
						}
					}
					n.printPeerStore()
					go n.ConnectPeers()
				}
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
