// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package routing

import (
	"bitmark-network/util"
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"bitmark-network/messagebus"

	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

/*
const (
	announceInitial     = 2 * time.Minute      // startup delay before first send
	announceRebroadcast = 7 * time.Minute      // to prevent too frequent rebroadcasts
	announceInterval    = 11 * time.Minute     // regular polling time
	announceExpiry      = 5 * announceInterval // if no responses received within this time, delete the entry
)
*/
const (
	announceInitial     = 1 * time.Minute      // startup delay before first send
	announceRebroadcast = 2 * time.Minute      // to prevent too frequent rebroadcasts
	announceInterval    = 4 * time.Minute      // regular polling time
	announceExpiry      = 3 * announceInterval // if no responses received within this time, delete the entry
)

type announcer struct {
	log *logger.L
}

// initialise the announcer
func (ann *announcer) initialise() error {

	log := logger.New("announcer")
	ann.log = log
	log.Info("initialising…")

	return nil
}
func printAnnounce(listener []byte) string {
	maAddrs := Addrs{}
	err := proto.Unmarshal(listener, &maAddrs)
	if err != nil {
		return ""
	}
	printListeners := ""
	for idx, straddr := range util.ByteAddrsToString(maAddrs.Address) {
		if 0 == idx {
			printListeners = straddr
		} else {
			printListeners = fmt.Sprintf("%s%s\n", printListeners, straddr)
		}
	}

	return printListeners
}

// wait for incoming requests, process them and reply
func (ann *announcer) Run(args interface{}, shutdown <-chan struct{}) {

	log := ann.log

	log.Info("starting… routing routine")

	queue := messagebus.Bus.Announce.Chan()

	delay := time.After(announceInitial)
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			switch item.Command {
			case "reconnect":
				determineConnections(log)
				log.Infof("-><- reconnect")
			case "updatetime":
				setPeerTimestamp(item.Parameters[0], time.Now())
				log.Infof("-><- updatetime id:%s", string(item.Parameters[0]))
			case "addpeer":
				//TODO: Make sure the timestamp is from external message or  local timestamp
				timestamp := binary.BigEndian.Uint64(item.Parameters[2])
				var listeners Addrs
				proto.Unmarshal(item.Parameters[1], &listeners)
				addrs := util.GetMultiAddrsFromBytes(listeners.Address)
				addPeer(item.Parameters[0], addrs, timestamp)
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warnf("IDFromByte Error:", err)
					continue loop
				}
				log.Infof("-><- add peer : %x  listener: %x  timestamp: %d", id, printAnnounce(item.Parameters[1]), timestamp)
			case "self":
				var lsners Addrs
				proto.Unmarshal(item.Parameters[1], &lsners)
				addrs := util.GetMultiAddrsFromBytes(lsners.Address)
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if nil == err {
					log.Infof("-><-  add self announce data: %v  listener: %s", id, printAnnounce(item.Parameters[1]))
					setSelf(item.Parameters[0], addrs)
				}
			default:
			}
		case <-delay: // Periodically Announce Self
			delay = time.After(announceInterval)
			log.Info("Announce Interval Timeup")
			ann.process()
		}
	}
}

// process the annoucement and return response to client
func (ann *announcer) process() {

	log := ann.log

	log.Debug("process starting…")

	globalData.Lock()
	defer globalData.Unlock()

	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	// TODO: Need to Add RPC
	/*
		if globalData.rpcsSet {
			log.Debugf("send rpc: %x", globalData.fingerprint)
			messagebus.Bus.Broadcast.Send("rpc", globalData.fingerprint[:], globalData.rpcs, timestamp)
		}
	*/
	if globalData.peerSet {
		log.Debugf("send peer: %x", globalData.peerID)
		byteMessage, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(globalData.listeners)})
		if nil == err {
			messagebus.Bus.P2P.Send("peer", globalData.peerID, byteMessage, timestamp)
		}
	}
	// TODO: Need to Add RPC
	//expireRPC()
	expirePeer(log)

	if globalData.treeChanged {
		determineConnections(log)
		globalData.treeChanged = false
	}
}

func determineConnections(log *logger.L) {
	if nil == globalData.thisNode {
		log.Errorf("determineConnections called to early")
		return // called to early
	}

	// locate this node in the tree
	_, index := globalData.peerTree.Search(globalData.thisNode.Key())
	count := globalData.peerTree.Count()
	id, err := peerlib.IDFromBytes(globalData.peerID)
	if err != nil {
		log.Warnf("determineConnections:Parse Node ID Error:", err)
	}
	log.Infof("This Node index: %d  tree: %d  peerID: %v", index, count, id.String())

	// various increment values
	e := count / 8
	q := count / 4
	h := count / 2

	jump := 3      // to deal with N3/P3 and too few nodes
	if count < 4 { // if insufficient
		jump = 1 // just duplicate N1/P1
	}

	names := [11]string{
		"N1",
		"N3",
		"X1",
		"X2",
		"X3",
		"X4",
		"X5",
		"X6",
		"X7",
		"P1",
		"P3",
	}

	// compute all possible offsets
	// if count is too small then there will be duplicate offsets
	var n [11]int
	n[0] = index + 1             // N1 (+1)
	n[1] = index + jump          // N3 (+3)
	n[2] = e + index             // X⅛
	n[3] = q + index             // X¼
	n[4] = q + e + index         // X⅜
	n[5] = h + index             // X½
	n[6] = h + e + index         // X⅝
	n[7] = h + q + index         // X¾
	n[8] = h + q + e + index     // X⅞
	n[9] = index + count - 1     // P1 (-1)
	n[10] = index + count - jump // P3 (-3)

	u := -1
deduplicate:
	for i, v := range n {
		if v == index || v == u {
			continue deduplicate
		}
		u = v
		if v >= count {
			v -= count
		}
		node := globalData.peerTree.Get(v)
		if nil != node {
			peer := node.Value().(*peerEntry)
			if nil != peer {
				id, err := peerlib.IDFromBytes(peer.peerID)
				if err != nil {
					log.Infof("announcer: conver byte to ID faile, ID:%v", id)
				}
				log.Infof("announcer.go determine connection Peer ID in byte %v", id)

				pbAddr := util.GetBytesFromMultiaddr(peer.listeners)
				byteMsg, err := proto.Marshal(&Addrs{Address: pbAddr})
				if nil == err {
					messagebus.Bus.P2P.Send(names[i], peer.peerID, byteMsg)
					showIDFromByte(peer.peerID)
				}
			}
		}
	}
}

func expirePeer(log *logger.L) {
	now := time.Now()
	nextNode := globalData.peerTree.First()
scan_nodes:
	for node := nextNode; nil != node; node = nextNode {

		peer := node.Value().(*peerEntry)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if bytes.Equal(globalData.peerID, peer.peerID) {
			continue scan_nodes
		}
		log.Debugf("public key: %x timestamp: %s", peer.peerID, peer.timestamp.Format(timeFormat))
		if peer.timestamp.Add(announceExpiry).Before(now) {
			globalData.peerTree.Delete(key)
			globalData.treeChanged = true
			// TODO: Send to P2P to Expire
			//messagebus.Bus.Connector.Send("@D", peer.peerID, peer.listeners) //@D means: @->Internal Command, D->delete
			log.Infof("Peer Expired! public key: %x timestamp: %s is removed", peer.peerID, peer.timestamp.Format(timeFormat))
		}
	}
}
