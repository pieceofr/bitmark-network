// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package routing

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

//type pubkey []byte
type peerIDkey []byte
type peerEntry struct {
	peerID    []byte
	listeners []byte
	timestamp time.Time // last seen time
}

// string - conversion fro fmt package
func (p peerEntry) String() string {
	v4, v6 := util.PackedConnection(p.listeners).Unpack46()
	return fmt.Sprintf("%x @ %q %q - %s", p.peerID, v4, v6, p.timestamp.Format(timeFormat))
}

// SetPeer - called by the peering initialisation to set up this
// node's announcement data
func SetPeer(peerID []byte, listeners []byte) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.peerSet {
		return fault.ErrAlreadyInitialised
	}
	globalData.peerID = peerID
	globalData.listeners = listeners
	globalData.peerSet = true

	addPeer(peerID, listeners, uint64(time.Now().Unix()))

	globalData.thisNode, _ = globalData.peerTree.Search(peerIDkey(peerID))
	determineConnections(globalData.log)

	return nil
}

// isPeerExpiredFromTime - is peer expired from time
func isPeerExpiredFromTime(timestamp time.Time) bool {
	return timestamp.Add(announceExpiry).Before(time.Now())
}

// AddPeer - add a peer announcement to the in-memory tree
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func AddPeer(peerID []byte, listeners []byte, timestamp uint64) bool {
	globalData.Lock()
	rc := addPeer(peerID, listeners, timestamp)
	globalData.Unlock()
	return rc
}

// internal add a peer announcement, hold lock before calling
func addPeer(peerID []byte, listeners []byte, timestamp uint64) bool {
	ts := resetFutureTimestampToNow(timestamp)
	if isPeerExpiredFromTime(ts) {
		return false
	}

	peer := &peerEntry{
		peerID:    peerID,
		listeners: listeners,
		timestamp: ts,
	}

	if node, _ := globalData.peerTree.Search(peerIDkey(peerID)); nil != node {
		peer := node.Value().(*peerEntry)

		if ts.Sub(peer.timestamp) < announceRebroadcast {
			return false
		}
	}

	// add or update the timestamp in the tree
	recordAdded := globalData.peerTree.Insert(peerIDkey(peerID), peer)

	globalData.log.Debugf("added: %t  nodes in the peer tree: %d", recordAdded, globalData.peerTree.Count())

	// if adding this nodes data
	if bytes.Equal(globalData.peerID, peerID) {
		return false
	}

	if recordAdded {
		globalData.treeChanged = true
	}

	return true
}

// resetFutureTimestampToNow - reset future timestamp to now
func resetFutureTimestampToNow(timestamp uint64) time.Time {
	ts := time.Unix(int64(timestamp), 0)
	now := time.Now()
	if now.Before(ts) {
		return now
	}
	return ts
}

// GetNext - fetch the data for the next node in the ring for a given public key
func GetNext(publicKey []byte) ([]byte, []byte, time.Time, error) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.peerTree.Search(peerIDkey(publicKey))
	if nil != node {
		node = node.Next()
	}
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node {
		return nil, nil, time.Now(), fault.ErrInvalidPublicKey
	}
	peer := node.Value().(*peerEntry)
	return peer.peerID, peer.listeners, peer.timestamp, nil
}

// GetRandom - fetch the data for a random node in the ring not matching a givpubkeyen public key
func GetRandom(peerID []byte) ([]byte, []byte, time.Time, error) {
	globalData.Lock()
	defer globalData.Unlock()

retry_loop:
	for tries := 1; tries <= 5; tries += 1 {
		max := big.NewInt(int64(globalData.peerTree.Count()))
		r, err := rand.Int(rand.Reader, max)
		if nil != err {
			continue retry_loop
		}

		n := int(r.Int64()) // 0 … max-1

		node := globalData.peerTree.Get(n)
		if nil == node {
			node = globalData.peerTree.First()
		}
		if nil == node {
			break retry_loop
		}
		peer := node.Value().(*peerEntry)
		if bytes.Equal(peer.peerID, globalData.peerID) || bytes.Equal(peer.peerID, peerID) {
			continue retry_loop
		}
		return peer.peerID, peer.listeners, peer.timestamp, nil
	}
	return nil, nil, time.Now(), fault.ErrInvalidPublicKey
}

// SendRegistration - send a peer registration request to a client channel
func SendRegistration(client zmqutil.ClientIntf, fn string) error {
	chain := mode.ChainName()

	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	return client.Send(fn, chain, globalData.peerID, globalData.listeners, timestamp)
}

// Compare - public key comparison for AVL interface
func (p peerIDkey) Compare(q interface{}) int {
	return bytes.Compare(p, q.(peerIDkey))
}

// String - public key string convert for AVL interface
func (p peerIDkey) String() string {
	return fmt.Sprintf("%x", []byte(p))
}

// setPeerTimestamp - set the timestamp for the peer with given public key
func setPeerTimestamp(publicKey []byte, timestamp time.Time) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.peerTree.Search(peerIDkey(publicKey))
	log := globalData.log
	if nil == node {
		log.Errorf("The peer with public key %x is not existing in peer tree", publicKey)
		return
	}

	peer := node.Value().(*peerEntry)
	peer.timestamp = timestamp
}
