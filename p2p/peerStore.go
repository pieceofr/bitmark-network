package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

// addPeer to PeerStore
func (n *Node) addPeer(peerInfo *peer.AddrInfo) {
	for _, addr := range peerInfo.Addrs {
		n.Host.Peerstore().AddAddr(peerInfo.ID, addr, peerstore.ConnectedAddrTTL)
	}
}

// RemovePeer to PeerStore
func (n *Node) RemovePeer(peerInfo *peer.AddrInfo) {
	for _, peerID := range n.Host.Peerstore().Peers() {
		if peerID == peerInfo.ID {
			//n.Host.Peerstore().UpdateAddrs(peerInfo.ID, 0*time.Second)
		}
	}

}
