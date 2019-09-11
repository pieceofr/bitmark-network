package p2p

import (
	"bufio"

	"github.com/bitmark-inc/logger"
	"github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-peer"
)

//NodeStreamHandler for  node to handle stream
type peerStreamHandler struct {
	log        *logger.L
	Stream     network.Stream
	ReadWriter *bufio.ReadWriter
	ID         peer.ID
	Shutdown   chan<- struct{}
}

//Setup setup Handler
func (h *peerStreamHandler) Setup(id peer.ID, log *logger.L) {
	h.ID = id
	h.log = log
	log.Infof("New Stream Handler:%d\n\n", id)
}

// Handler  for streamHandler
func (h *peerStreamHandler) Handler(s network.Stream) {
	h.Stream = s
	go h.Reciever()
	go h.Sender()
	h.log.Infof("Start a new stream! ID:%d  direction:%d\n", h.ID, s.Stat().Direction)
}

//Reciever for NodeStreamHandler
func (h *peerStreamHandler) Reciever() {
	h.log.Infof("---Handler-%d Reciever Start---", h.ID)
	for {
		item, err := h.ReadWriter.ReadString('\n')
		if err != nil {
			h.log.Error(err.Error())
			continue
		}
		h.log.Infof("#Reader%d read: %v\n", h.ID, item)
	}
}

//Sender for NodeStreamHandler
func (h *peerStreamHandler) Sender() {
	h.log.Infof("---Handler-%d Sender Start---", h.ID)
}

func (h *peerStreamHandler) CloseStream() error {
	if err := h.Stream.Close(); err != nil {
		h.log.Errorf("Close Stream ID:%s Error:", h.ID, err)
		return err
	}
	h.log.Infof("---Handler-%d Close Stream---", h.ID)
	return nil
}
