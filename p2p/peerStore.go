package p2p

import (
	"fmt"

	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// addPeer to PeerStore
func (n *Node) addPeer(id peer.ID, peer ma.Multiaddr) {
	n.Lock()
	n.Host.Peerstore().AddAddr(id, peer, peerstore.ConnectedAddrTTL)
	n.Unlock()

	for index, id := range n.Host.Peerstore().PeersWithAddrs() {
		infoAddrs := n.Host.Peerstore().Addrs(id)
		addrsString := ""
		for _, addr := range infoAddrs {
			addrsString = fmt.Sprintf("%s-%s", addrsString, addr)
		}
		n.log.Infof("addPeer[%d]: ID:%v\nAddrs:%s", index, id, addrsString)
	}
}
