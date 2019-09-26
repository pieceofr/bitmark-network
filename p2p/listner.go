package p2p

import (
	"bitmark-network/announce"
	"bitmark-network/util"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const maxBytesRecieve = 2000

//ListenHandler is a host Listening  handler
type ListenHandler struct {
	ID  string
	log *logger.L
}

//NewListenHandler return a NewListenerHandler
func NewListenHandler(ID string, log *logger.L) ListenHandler {
	return ListenHandler{ID: ID, log: log}
}

func (l *ListenHandler) handleStreamX(stream network.Stream) {
	log := l.log
	log.Info("--- Start A New stream --")
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
loop:
	for {
		//data := make([]byte, maxBytesRecieve)
		data, _, err := rw.ReadLine()
		if err != nil {
			log.Error(err.Error())
			continue loop
		}
		log.Infof("len of rRead Data:%d", len(data))
		if len(data) < 1 {
			listenerSendError(rw, errors.New("Invalid Request Data"), log)
			log.Error("length of byte recieved is less than 1")
		}

		var reqMsg P2PMessage
		UnmarshalErr := proto.Unmarshal(data, &reqMsg)
		if UnmarshalErr != nil {
			listenerSendError(rw, errors.New("Invalid Request Data"), log)
			log.Error("Unmarshal Request Message Error")
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
				log.Errorf("Get a random  node Error %v", err)
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
				stream.Reset()
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
		log.Errorf("Send Error Message Error:%v", err)
	}
	if log != nil {
		log.Errorf("<-- Error Message:%v", err)
	}
	sender.Flush()
}

func (l *ListenHandler) handleStream(stream network.Stream) {
	log := l.log
	log.Info("--- Start A New stream --")
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer stream.Reset()
loop:
	for {
		//TODO: Ia any better way to reciece data?
		req := make([]byte, maxBytesRecieve)
		lenByte, err := rw.Read(req)
		log.Info(fmt.Sprintf("%s RECIEVE:\x1b[32m%d\x1b[0m> ", "listener", lenByte))
		if err != nil {
			log.Error(err.Error())
			continue loop
		}
		if lenByte < 1 {
			log.Error("length of byte recieved is less than 1")
		}
		reqMessageUnPacked := P2PMessage{}
		proto.Unmarshal(req, &reqMessageUnPacked)
		if len(reqMessageUnPacked.Data) == 0 {
			return
		}
		printP2PMessage(reqMessageUnPacked, log)
		chain := string(reqMessageUnPacked.Data[0])
		fn := string(reqMessageUnPacked.Data[1])
		parameters := reqMessageUnPacked.Data[2:]

		switch fn {
		case "R":
			reqID, err := peerlib.IDFromBytes(parameters[0])
			if err != nil {
				log.Infof(err.Error())
				continue loop
			}
			var reqListener Addrs
			errAddr := proto.Unmarshal(parameters[1], &reqListener)
			if errAddr != nil {
				log.Error(errAddr.Error())
				continue loop
			}
			reqMaAddrs := util.GetMultiAddrsFromBytes(reqListener.Address)
			timestamp := binary.BigEndian.Uint64(parameters[2])
			log.Infof("chain:%s, fn:%s reqID:%s timestamp%d listeners:%s", chain, fn, reqID.String(), timestamp, util.PrintMaAddrs(reqMaAddrs))
			announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestam

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
			respData := P2PMessage{Data: "Hi There"}
			mockPacked, err := proto.Marshal(&mock)
			if err != nil {
				log.Error(err.Error())
				continue loop
			}
			_, err = rw.Write(append(mockPacked, '\r', '\n'))
			rw.Flush()
			if err != nil {
				log.Error(err.Error())
				continue loop
			}
		*/
	}
}

func printP2PMessage(msg P2PMessage, l *logger.L) {
	chain := string(msg.Data[0])
	fn := string(msg.Data[1])
	id, _ := peerlib.IDFromBytes(msg.Data[2])
	l.Info(fmt.Sprintf("%s RECIEVE: chain: \x1b[32m%s fn:\x1b[0m%s> id:%s", "listener", chain, fn, id.String()))
}
