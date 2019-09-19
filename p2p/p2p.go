package p2p

import (
	"bitmark-network/messagebus"
	"bitmark-network/util"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	p2pcore "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

// global data
var globalData Node
var bitmarkprotocol = "/bitmark/1.0.0"

// const
const (
	// domains
	domainLocal   = "nodes.rachael.bitmark"
	domainBitamrk = "nodes.test.bitmark.com"
	domainTest    = "nodes.test.bitmark.com"
	//  time interval
	nodeInitial  = 5 * time.Second // startup delay before first send
	nodeInterval = 1 * time.Minute // regular polling time
)

var (
	// muticastingTopic
	multicastingTopic = "/peer/announce/1.0.0"
	// stream protocols
	nodeProtocol = ma.ProtocolWithCode(ma.P_P2P).Name
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

// NodeType to inidcate a node is a servant or client
type NodeType int

const (
	// Servant acts as both server and client
	Servant NodeType = iota
	// Client acts as a client only
	Client
	// Server acts as a server only, not supported at first draft
	Server
)

//Node  A p2p node
type Node struct {
	NodeType       string
	Host           p2pcore.Host
	Announce       []ma.Multiaddr
	sync.RWMutex             // to allow locking
	log            *logger.L // logger
	MuticastStream *pubsub.PubSub
	PreferIPv6     bool
	PrivateKey     crypto.PrivKey
	// for background
	background *background.T
	// set once during initialise
	initialised bool
}

// Initialise initialize p2p module
func Initialise(configuration *Configuration) error {
	globalData.Lock()
	defer globalData.Unlock()
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}
	globalData.log = logger.New("p2p")
	globalData.log.Info("starting…")
	globalData.Setup(configuration)
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
				id, err := peerlib.IDFromBytes(messageOut.Parameters[0])
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
					log.Infof("Command:%v", item.Command)
					peerID, err := peerlib.IDFromBytes(item.Parameters[0])
					log.Infof("N1 PeerID%s", peerID.String())
					if err != nil {
						n.log.Errorf("Unmarshal peer ID Error:%x", item.Parameters[0])
						goto loop
					}
					pbPeerAddrs := Addrs{}
					proto.Unmarshal(item.Parameters[1], &pbPeerAddrs)
					maAddrs := util.GetMultiAddrsFromBytes(pbPeerAddrs.Address)

					info, err := peer.AddrInfoFromP2pAddr(maAddrs[0])
					if err != nil {
						log.Warn(err.Error())
						continue loop
					}
					log.Infof("Connecting to Address %s", info.Addrs[0].String())
					n.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

					// Start a stream with the destination.
					// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
					s, err := n.Host.NewStream(context.Background(), info.ID, "/chat/1.0.0")
					if err != nil {
						log.Warn(err.Error())
						continue loop
					}
					// Create a thread to read and write data.
					var shandler basicStream

					shandler.ID = fmt.Sprintf("%s", n.Host.ID())
					shandler.handleStream(s)
					/*
						s, err := n.Host.NewStream(context.Background(), peerID, "/chat/1.0.0")
						if err != nil {
							n.log.Errorf("connect to peerID:%s error:%s addrs:%s", peerID, err, util.PrintMaAddrs(maAddrs))
						} else {

								var handleStream nodeStreamHandler
								handleStream.setup(&n.Host, n.log)
								handleStream.Handler(s)
					*/
				}
				n.printPeerStore()
				//	go n.connectPeers()

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
