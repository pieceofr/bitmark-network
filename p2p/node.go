package p2p

import (
	"bitmark-network/messagebus"
	"bitmark-network/util"
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	p2pcore "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	protocol "github.com/libp2p/go-libp2p-protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

//Setup setup a node
func (n *Node) Setup(configuration *Configuration) error {
	globalData.NodeType = configuration.NodeType
	globalData.PreferIPv6 = configuration.PreferIPv6
	maAddrs := IPPortToMultiAddr(configuration.Listen)
	prvKey, err := DecodeHexToPrvKey([]byte(configuration.PrivateKey)) //Hex Decoded binaryString
	if err != nil {
		globalData.log.Error(err.Error())
		panic(err)
	}
	n.PrivateKey = prvKey
	n.Host = NewHost(configuration.NodeType, maAddrs, n.PrivateKey)
	n.setAnnounce(configuration.Announce)
	// Start to listen to p2p stream
	go n.listen()
	// Create a Multicasting route
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
func NewHost(nodetype string, listenAddrs []ma.Multiaddr, prvKey crypto.PrivKey) p2pcore.Host {
	options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New)}
	if "client" != nodetype {
		options = append(options, libp2p.ListenAddrs(listenAddrs...))
	}
	newHost, err := libp2p.New(context.Background(), options...)
	if err != nil {
		panic(err)
	}
	globalData.log.Infof("New Host is created ID:%v", newHost.ID())
	for _, a := range newHost.Addrs() {
		globalData.log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
	}
	return newHost
}

//setAnnounce: Set Announce address in Routing
func (n *Node) setAnnounce(announceAddrs []string) {
	maAddrs := IPPortToMultiAddr(announceAddrs)
	fullAddr := announceMuxAddr(maAddrs, nodeProtocol, n.Host.ID())
	byteMessage, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(fullAddr)})
	param0, idErr := n.Host.ID().Marshal()

	if nil == err && nil == idErr {
		messagebus.Bus.Announce.Send("self", param0, byteMessage)
	}
}

// listen  connect to other node , this is a blocking operation
func (n *Node) listen() error {
	handler := nodeStreamHandler{}
	handler.setup(&n.Host, n.log)
	n.Host.SetStreamHandler(protocol.ID(nodeProtocol), handler.Handler)
	<-make(chan struct{})
	return nil
}
