package p2p

import (
	"bitmark-network/messagebus"
	"bitmark-network/util"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
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
	nodeProtocol      = "p2p"
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
	maAddrs := configListenToMultiAddrs(configuration.Listen)
	globalData.Host = NewHost(configuration.NodeType, maAddrs, configuration.PrivateKey)
	n.setAnnounce(configuration.Announce)
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
func NewHost(nodetype string, listenAddrs []ma.Multiaddr, privateKey string) p2pcore.Host {
	prvKey, err := DecodeHexToPrvKey([]byte(privateKey)) //Hex Decoded binaryString
	if err != nil {
		globalData.log.Error(err.Error())
		panic(err)
	}
	// For Client Node
	newHost, err := libp2p.New(
		context.Background(),
		libp2p.Identity(prvKey),
		libp2p.Security(tls.ID, tls.New),
	)
	if err != nil {
		panic(err)
	}
	if "client" == nodetype {
		globalData.log.Infof("create a client host ID:%v", newHost.ID())
		return newHost
	}
	// For Servant Node
	newHost, err = libp2p.New(
		context.Background(),
		libp2p.Identity(prvKey),
		libp2p.Security(tls.ID, tls.New),
		libp2p.ListenAddrs(listenAddrs...),
	)
	if err != nil {
		panic(err)
	}
	globalData.log.Infof("create a servant host ID:%v", newHost.ID())
	for _, a := range newHost.Addrs() {
		globalData.log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
	}
	return newHost
}

//setAnnounce: Set Announce address in Routing
func (n *Node) setAnnounce(announceAddrs []string) {
	maAddrs := configListenToMultiAddrs(announceAddrs)
	byteMessage, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(maAddrs)})
	param0, idErr := n.Host.ID().Marshal()

	if nil == err && nil == idErr {
		messagebus.Bus.Announce.Send("self", param0, byteMessage)
	}
}

// ConnectPeers connect to all peers in host peerstore
func (n *Node) ConnectPeers() {
	for _, peerID := range n.Host.Peerstore().Peers() {
		peerInfo := n.Host.Peerstore().PeerInfo(peerID)
		n.ConnectStream(&peerInfo)
	}
}

// ConnectStream  connect to other node , this is a blocking operation
func (n *Node) ConnectStream(peer *peer.AddrInfo) error {
	n.Host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.ConnectedAddrTTL)
	s, err := n.Host.NewStream(context.Background(), peer.ID, nodeProtocol)
	if err != nil {
		return err
	}
	var handleStream peerStreamHandler
	handleStream.Setup(peer.ID, n.log)
	handleStream.Handler(s)
	return nil
}
