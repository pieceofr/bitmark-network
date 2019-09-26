package announce

import (
	"bitmark-network/avl"
	"bitmark-network/util"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/logger"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	var logLevel map[string]string
	logLevel = make(map[string]string, 0)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "announce_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.log = logger.New("nodes")
	os.Exit(m.Run())
}

func TestGetNode(t *testing.T) {
	log := globalData.log
	tree := mockTree()

	for i := 0; i < tree.Count(); i++ {
		node := tree.Get(i)
		peer := node.Value().(*peerEntry)
		log.Infof("%v", peer.peerID)
	}
}
func TestRandom(t *testing.T) {
	globalData.peerTree = mockTree()
	peers := mockPeer()

	id, _, _, err := GetRandom(peers[0].peerID)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Input ID:", peers[0].peerID.String(), "Random ID:", id.String())
	assert.NotEqual(t, peers[0].peerID.String(), id.String())
}

func TestStorePeers(t *testing.T) {
	fmt.Println("TestStorePeers")
	curPath := os.Getenv("PWD")
	peerfile := path.Join(curPath, "peers.store")
	// domain from bind9
	err := Initialise("nodes.rachael.bitmark", peerfile)
	globalData.peerTree = mockTree()
	assert.NoError(t, err, "announce initialized error")
	storePeers(peerfile)
	assert.NoError(t, err, "announce backupPeers error")
}
func TestReadPeers(t *testing.T) {
	fmt.Println("TestReadPeers")
	curPath := os.Getenv("PWD")
	peerfile := path.Join(curPath, "peers.store")
	fmt.Println("Read File Path:", peerfile)
	var peers PeerList
	readin, err := ioutil.ReadFile(peerfile)
	assert.NoError(t, err, "TestReadPeers:readFile Error")
	proto.Unmarshal(readin, &peers)
	for _, peer := range peers.Peers {
		addrList := util.ByteAddrsToString(peer.Listeners.Address)
		id, _ := peerlib.IDFromBytes(peer.PeerID)
		fmt.Printf("peerID:%s, listener:%v timestamp:%d\n", id, addrList, peer.Timestamp)
	}
}

func mockTree() *avl.Tree {
	peerTree := avl.New()
	for _, peer := range mockPeer() {
		key := peer.peerID
		peerTree.Insert(peerIDkey(key), &peer)
	}
	//peerTree.Print(false)
	return peerTree
}

func mockPeer() []peerEntry {
	mockpeers := []peerEntry{}
	id1, err := peerlib.IDB58Decode("12D3KooWPUg7Gx4CaSX2r591XNZzDRgx39zy2RCkAYggh5vBb7vT")
	if err != nil {
		panic(err)
	}

	id2, _ := peerlib.IDB58Decode("12D3KooWCvzWT8L6HTtAwac5Vr7RzLitbyZkJ9T9fnikWonnS1DR")
	id3, _ := peerlib.IDB58Decode("12D3KooWN7RiKsKMJ7eaN8iytPhN44tnmwsKT5b2qRRX9dG6nb8G")
	id4, _ := peerlib.IDB58Decode("12D3KooWEyjhqbupsDryotHfHhcUfNF1mMyhGEFTaRMaPr6i28iw")
	id5, _ := peerlib.IDB58Decode("12D3KooWEyjhqbupsDryotHfHhcUfNF1mMyhGEFTaRMaPr6i28iw")
	ma1, _ := ma.NewMultiaddr("/ip4/172.17.0.1/tcp/12137/ipfs/12D3KooWPUg7Gx4CaSX2r591XNZzDRgx39zy2RCkAYggh5vBb7vT")
	ma2, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/2136/ipfs/12D3KooWCvzWT8L6HTtAwac5Vr7RzLitbyZkJ9T9fnikWonnS1DR")
	ma3, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/12138/ipfs/12D3KooWN7RiKsKMJ7eaN8iytPhN44tnmwsKT5b2qRRX9dG6nb8G")
	ma4, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/12139/ipfs/12D3KooWEyjhqbupsDryotHfHhcUfNF1mMyhGEFTaRMaPr6i28iw")

	mockpeers = append(mockpeers, peerEntry{
		peerID:    id1,
		listeners: []ma.Multiaddr{ma1},
	})
	mockpeers = append(mockpeers, peerEntry{
		peerID:    id2,
		listeners: []ma.Multiaddr{ma2},
	})
	mockpeers = append(mockpeers, peerEntry{
		peerID:    id3,
		listeners: []ma.Multiaddr{ma3},
	})
	mockpeers = append(mockpeers, peerEntry{
		peerID:    id4,
		listeners: []ma.Multiaddr{ma4},
	})
	mockpeers = append(mockpeers, peerEntry{
		peerID:    id5,
		listeners: []ma.Multiaddr{ma4},
	})
	return mockpeers
}
