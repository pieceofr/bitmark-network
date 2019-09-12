package p2p

import (
	"context"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	net "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/common/log"
)

func (n *Node) dialPeer(ctx context.Context, remoteListner *peer.AddrInfo) (net.Stream, error) {
	cctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()
	return n.Host.NewStream(cctx, remoteListner.ID, protocol.ID(nodeProtocol))
}

// ConnectPeers connect to all peers in host peerstore
func (n *Node) connectPeers() {

	for _, peerID := range n.Host.Peerstore().Peers() {
		peerInfo := n.Host.Peerstore().PeerInfo(peerID)

		if !n.isSameNode(peerInfo.Addrs) {
			s, err := n.dialPeer(context.Background(), &peerInfo)
			if err != nil {
				log.Errorf("dialPeer Error:%v", err)
			}
			var handleStream nodeStreamHandler
			handleStream.setupRemote(&n.Host, peerID)
			handleStream.Handler(s)
		}
	}
}

func (n *Node) isSameNode(addrs []ma.Multiaddr) bool {
	for _, cmpr := range addrs {
		for _, a := range n.Host.Addrs() {
			if strings.Contains(cmpr.String(), a.String()) {
				return true
			}
		}
	}
	return false
}
