package p2p

import (
	"fmt"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

//TODO: This function add address into the peer with the same id. Needs to take care of  IP changes
// addPeer to PeerStore
func (n *Node) addPeer(id peer.ID, peer ma.Multiaddr) {
	n.Lock()
	n.log.Infof("add peerstore:%s", peer.String())
	n.Host.Peerstore().AddAddr(id, peer, peerstore.ConnectedAddrTTL)
	n.Unlock()
}

func (n *Node) printPeerStore() {
	if len(n.Host.Peerstore().PeersWithAddrs()) == 0 {
		n.log.Warn("no peers in peerstore")
		return
	}
	for index, id := range n.Host.Peerstore().PeersWithAddrs() {
		infoAddrs := n.Host.Peerstore().Addrs(id)
		addrsString := ""
		for _, addr := range infoAddrs {
			addrsString = fmt.Sprintf("%s-%s", addrsString, addr)
		}
		n.log.Infof("Peerstore[%d]: ID:%v  time:%d \nAddrs:%s", index, id, time.Now().Unix(), addrsString)
	}
}
