package p2p

import (
	"bufio"
	"time"

	"github.com/bitmark-inc/logger"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prometheus/common/log"
)

// nodeStreamHandler for P2P
type nodeStreamHandler struct {
	Host       *libp2pcore.Host
	Stream     network.Stream
	ReadWriter *bufio.ReadWriter
	RemoteID   peer.ID
	log        *logger.L
	Shutdown   chan<- struct{}
}

func (h *nodeStreamHandler) setup(host *libp2pcore.Host) {
	h.Host = host
	log.Infof("New Stream Handler")
}
func (h *nodeStreamHandler) setupRemote(host *libp2pcore.Host, id peer.ID) {
	h.Host = host
	h.RemoteID = id
	log.Infof("New Stream Handler")
}

// Handler  for streamHandler
func (h *nodeStreamHandler) Handler(s network.Stream) {
	h.Stream = s
	h.ReadWriter = bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	shutdown := make(chan struct{})
	h.Shutdown = shutdown
	go h.Reciever()
	go h.Sender(shutdown)
	log.Debugf("Start a new stream! ID:%s  direction:%d\n", h.RemoteID, s.Stat().Direction)
}

//Reciever for NodeStreamHandler
func (h *nodeStreamHandler) Reciever() {
	log.Infof("---Handler Reciever Start- RemoteID:%s--", h.RemoteID)
	em, err := (*h.Host).EventBus().Emitter(new(P2PMessage))
	if err != nil {
		panic(err)
	}
	defer em.Close()
	for {
		str, err := h.ReadWriter.ReadString('\n')
		if err != nil {
			log.Error(err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		log.Infof("#Reader%d read: %v\n", h.RemoteID, str)
	}
}

//Sender for NodeStreamHandler
func (h *nodeStreamHandler) Sender(shutdown <-chan struct{}) {
	log.Infof("---Handler Sender Start- RemoteID:%s--", h.RemoteID)
}
