package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/logger"
	libp2p "github.com/libp2p/go-libp2p"
	p2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

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
const (
	nodeInitial       = 5 * time.Second // startup delay before first send
	nodeInterval      = 1 * time.Minute // regular polling time
	multicastingTopic = "/peer/announce/1.0.0"
	nodeProtocol      = "/p2p"
)

//Node  A p2p node
type Node struct {
	PublicKey string
	//StreamHandle *StreamHandling
	NodeType       string
	Host           p2pcore.Host
	sync.RWMutex             // to allow locking
	log            *logger.L // logger
	MuticastStream *pubsub.PubSub
	// for background
	background *background.T
	// set once during initialise
	initialised bool
}

//Setup setup a node
func (n *Node) Setup(configuration *Configuration) error {
	globalData.NodeType = configuration.NodeType
	globalData.PublicKey = configuration.PublicKey
	globalData.Host = NewHost(configuration.NodeType, configuration.Listen, configuration.PrivateKey)
	// Create a Multicasting function
	ps, err := pubsub.NewGossipSub(context.Background(), n.Host)
	if err != nil {
		panic(err)
	}
	n.MuticastStream = ps
	sub, err := n.MuticastStream.Subscribe(multicastingTopic)
	go n.SubHandler(context.Background(), sub)
	globalData.initialised = true
	return nil
}

// NewHost create a NewHost according to nodetype
func NewHost(nodetype string, listenAddr []string, privateKey string) p2pcore.Host {
	globalData.log.Infof("Private key:%x", []byte(privateKey))
	prvKey, err := DecodePrvKey([]byte(privateKey)) //Hex Decoded binaryString
	if err != nil {
		globalData.log.Error(err.Error())
		panic(err)
	}
	// For Client Node
	if "client" == nodetype {
		newHost, err := libp2p.New(
			context.Background(),
			libp2p.Identity(prvKey),
			libp2p.Security(tls.ID, tls.New),
		)
		if err != nil {
			panic(err)
		}
		return newHost
	}
	// For Servant Node
	var hostListenAddress ma.Multiaddr
	hostListenAddress, err = NewListenMultiAddr(listenAddr)
	newHost, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(hostListenAddress),
		libp2p.Identity(prvKey),
		libp2p.Security(tls.ID, tls.New),
	)
	if err != nil {
		panic(err)
	}
	for _, addr := range newHost.Addrs() {
		globalData.log.Infof("New Host Address: %s/%v/%s\n", addr, "p2p", newHost.ID())
	}
	return newHost
}

// Connect  connect to other node , this is a blocking operation
func (n *Node) Connect(peer peer.AddrInfo) error {
	err := n.Host.Connect(context.Background(), peer)
	if err != nil {
		return err
	}
	for _, addr := range peer.Addrs {
		n.Host.Peerstore().AddAddr(peer.ID, addr, peerstore.ConnectedAddrTTL)
	}
	return nil
}
