package p2p

import (
	"bitmark-network/messagebus"
	"bitmark-network/util"
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
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
	n.NewHost(configuration.NodeType, maAddrs, n.PrivateKey)
	n.setAnnounce(configuration.Announce)
	// Start to listen to p2p stream
	//go n.listen()
	go n.listenTemp(configuration.Announce)
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
func (n *Node) NewHost(nodetype string, listenAddrs []ma.Multiaddr, prvKey crypto.PrivKey) error {
	options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New)}
	if "client" != nodetype {
		options = append(options, libp2p.ListenAddrs(listenAddrs...))
	}
	newHost, err := libp2p.New(context.Background(), options...)
	if err != nil {
		panic(err)
	}
	n.Host = newHost
	globalData.log.Infof("New Host is created ID:%v", newHost.ID())
	for _, a := range newHost.Addrs() {
		globalData.log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
	}
	return nil
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
	n.Host.SetStreamHandler("/chat/1.0.0", handler.Handler)
	<-make(chan struct{})
	return nil
}

func (n *Node) listenTemp(announceAddrs []string) {
	maAddrs := IPPortToMultiAddr(announceAddrs)
	var shandler SimpleStream

	shandler.ID = fmt.Sprintf("%s", n.Host.ID())
	n.log.Infof("A servant is listen to %s", maAddrs[0].String())

	n.Host.SetStreamHandler("/chat/1.0.0", shandler.handleStream)

	// Hang forever
	<-make(chan struct{})
}
