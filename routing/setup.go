package routing

import (
	"bitmark-network/avl"
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

type routerData struct {
	sync.RWMutex // to allow locking
	// logger
	log *logger.L
	// PeerEntry Info
	// this node's packed annoucements
	peerID    []byte
	listeners []byte
	// bootstrap
	nodesLookup nodesLookup
	// tree of nodes available
	peerTree    *avl.Tree
	thisNode    *avl.Node // this node's position in the tree
	treeChanged bool      // tree was changed
	peerSet     bool
	peerFile    string
	// data for thread
	ann announcer
	// for background
	background *background.T
	// set once during initialise
	initialised bool
}

// global data
var globalData routerData

// format for timestamps
const timeFormat = "2006-01-02 15:04:05"

// Initialise - set up the announcement system
// pass a fully qualified domain for root node list
// or empty string for no root nodes
func Initialise(nodesDomain, peerFile string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("announce")
	globalData.log.Info("starting…")

	globalData.peerTree = avl.New()
	globalData.thisNode = nil
	globalData.treeChanged = false

	globalData.peerSet = false
	globalData.peerFile = peerFile

	globalData.log.Info("start restoring peer data…")
	if _, err := restorePeers(globalData.peerFile); err != nil {

		globalData.log.Errorf("fail to restore peer data: %s", err.Error())
	}

	if err := globalData.nodesLookup.initialise(nodesDomain); nil != err {
		return err
	}

	if err := globalData.ann.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		&globalData.nodesLookup, &globalData.ann,
	}

	globalData.background = background.Start(processes, globalData.log)
	return nil
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	globalData.log.Info("start backing up peer data…")
	if err := backupPeers(globalData.peerFile); err != nil {
		globalData.log.Errorf("fail to backup peer data: %s", err.Error())
	}

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// MarshalText - convert fingerprint to little endian hex text
// TODO: ADD RPC
/*
func (fingerprint fingerprintType) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(fingerprint))
	buffer := make([]byte, size)
	hex.Encode(buffer, fingerprint[:])
	return buffer, nil
}
*/
