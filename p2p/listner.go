package p2p

import (
	"bitmark-network/announce"
	"bitmark-network/util"
	"bufio"
	"encoding/binary"
	"errors"
	"io/ioutil"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

//ListenHandler is a host Listening  handler
type ListenHandler struct {
	ID  string
	log *logger.L
}

//NewListenHandler return a NewListenerHandler
func NewListenHandler(ID string, log *logger.L) ListenHandler {
	return ListenHandler{ID: ID, log: log}
}

func (l *ListenHandler) handleStream(stream network.Stream) {
	log := l.log
	log.Info("--- Start A New stream --")
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
loop:
	for {
		data, err := ioutil.ReadAll(rw)
		if err != nil {
			listenerSendError(rw, err, log)
		}
		var reqMsg P2PMessage
		UnmarshalErr := proto.Unmarshal(data, &reqMsg)
		if UnmarshalErr != nil {
			continue loop
		}

		reqChain := string(reqMsg.Data[0])
		if len(data) < 2 {
			listenerSendError(rw, err, log)
		}

		if reqChain != mode.ChainName() {
			log.Errorf("invalid chain: actual: %q  expect: %s", reqChain, mode.ChainName())
			listenerSendError(rw, err, log)
			continue loop
		}
		fn := string(reqMsg.Data[1])
		parameters := reqMsg.Data[2:4]
		log.Debugf("received message: %q: %x", fn, parameters)
		//result := []byte{}
		switch fn {
		case "R": // params[id, listeners, timestamp]
			var reqID peerlib.ID
			var reqListener Addrs
			reqID, errID := peer.IDFromBytes(parameters[0])
			errAddr := proto.Unmarshal(parameters[1], &reqListener)
			if errID != nil || errAddr != nil {
				listenerSendError(rw, errors.New("Unmarshal error"), log)
			}
			timestamp := binary.BigEndian.Uint64(parameters[2])
			reqMaAddrs := util.GetMultiAddrsFromBytes(reqListener.Address)
			//TODO: Should not  directly us announce module ... Change the way
			announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestamp
			//TODO: Should not  directly us announce module ... Change the way
			randPeerID, randListeners, randTs, err := announce.GetRandom(reqID)
			if nil != err {
				listenerSendError(rw, err, log)
				continue loop
			}
			randIDPacked, idErr := randPeerID.Marshal()
			randListenerPackaed, addrErr := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(randListeners)})
			if idErr != nil || addrErr != nil {
				listenerSendError(rw, err, log)
				continue loop
			}
			randTsPacked := make([]byte, 8)
			binary.BigEndian.PutUint64(randTsPacked, uint64(randTs.Unix()))
			respData := [][]byte{[]byte(mode.ChainName()), []byte(fn), randIDPacked, randListenerPackaed, randTsPacked}
			reqMsgPacked, err := proto.Marshal(&P2PMessage{Data: respData})
			if err != nil {
				listenerSendError(rw, err, log)
				continue loop
			}
			_, err = rw.Write(reqMsgPacked)
		}
		/*
			if str != "\n" {
				fmt.Printf("%s RECIEVE:\x1b[32m%s\x1b[0m> ", l.ID, str)
			}
		*/
	}
}

func listenerSendError(sender *bufio.ReadWriter, err error, log *logger.L) {
	errorMessage := err.Error()
	_, wErr := sender.WriteString(errorMessage + "\n")
	if wErr != nil && log != nil {
		log.Infof("Send Error Message Error:%v", err)
	}
	if log != nil {
		log.Infof("<-- Error Message:%v", err)
	}
}
