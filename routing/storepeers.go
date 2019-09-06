// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package routing

import (
	"io/ioutil"

	proto "github.com/golang/protobuf/proto"
)

// NewPeerItem is to create a PeerItem from peerEntry
func NewPeerItem(peer *peerEntry) *PeerItem {
	if peer == nil {
		return nil
	}
	return &PeerItem{
		PeerID:    peer.peerID,
		Listeners: peer.listeners,
		Timestamp: uint64(peer.timestamp.Unix()),
	}
}

// backupPeers will backup all peers into a peer file
func backupPeers(peerFile string) error {
	if globalData.peerTree.Count() <= 2 {
		globalData.log.Info("no need to backup. peer nodes are less than two")
		return nil
	}
	var peers PeerList
	lastNode := globalData.peerTree.Last()
	node := globalData.peerTree.First()

	for node != lastNode {
		peer, ok := node.Value().(*peerEntry)
		if ok {
			p := NewPeerItem(peer)
			peers.Peers = append(peers.Peers, p)
		}
		node = node.Next()
	}
	// backup the last node
	peer, ok := lastNode.Value().(*peerEntry)
	if ok {
		p := NewPeerItem(peer)
		peers.Peers = append(peers.Peers, p)
	}
	out, err := proto.Marshal(&peers)
	if err != nil {
		globalData.log.Errorf("Failed to marshal peers protobuf:%v", err)
	}
	if err := ioutil.WriteFile(peerFile, out, 0600); err != nil {
		globalData.log.Errorf("Failed to write peers to a file:%v", err)
	}
	return nil
}

// restorePeers will backup peers from a peer file
func restorePeers(peerFile string) (PeerList, error) {
	var peers PeerList
	readin, err := ioutil.ReadFile(peerFile)
	if err != nil {
		globalData.log.Errorf("Failed to read peers from a file:%v", err)
	}
	proto.Unmarshal(readin, &peers)
	for _, peer := range peers.Peers {
		addPeer(peer.PeerID, peer.Listeners, peer.Timestamp)
	}
	return peers, nil
}
