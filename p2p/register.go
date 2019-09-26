package p2p

import (
	"bitmark-network/announce"
	"bitmark-network/messagebus"
	"bitmark-network/util"
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/prometheus/common/log"
)

func (n *Node) registerX(peerInfo *peerlib.AddrInfo) (*network.Stream, error) {
	s, err := n.Host.NewStream(context.Background(), peerInfo.ID, "p2pstream")
	n.log.Info("---- start to register ---")
	if err != nil {
		n.log.Warn(err.Error())
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	err = n.sendRegisterRequest(s, rw, n.Host.ID().Pretty())
	if err != nil {
		n.log.Errorf("sendRegisterRequest Error:%v", err)
		s.Close()
		return nil, err
	}
	return &s, nil
}

func (n *Node) sendRegisterRequest(stream network.Stream, rw *bufio.ReadWriter, who string) error {
	//TODO:Make dynamic get chain
	log := n.log
	chain := "local"
	fn := "R"
	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	addrsPacked, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(n.Announce)})
	id, idErr := n.Host.ID().Marshal()
	if err != nil || idErr != nil {
		return errors.New("parameter marshal error")
	}
	reqData := [][]byte{[]byte(chain), []byte(fn), id, addrsPacked, timestamp}
	reqMsgPacked, err := proto.Marshal(&P2PMessage{Data: reqData})
	n.log.Infof("len of reqData:%d", len(reqMsgPacked))
	if err != nil {
		return err
	}
	_, err = rw.Write(append(reqMsgPacked, '\r', '\n'))
	if err != nil {
		return err
	}
	rw.Flush()

	// wait for response
	data := make([]byte, maxBytesRecieve)
	dataLen, err := rw.Read(data)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if dataLen < 2 {
		return fmt.Errorf("register: %d receive expected at least 2", len(data))
	}

	var respMessage P2PMessage
	err = proto.Unmarshal(data, &respMessage)
	if nil != err {
		return fmt.Errorf("register: %v receive error: %s", n.Host.ID(), err)
	}
	n.log.Info(fmt.Sprintf("%s RECIEVE:\x1b[32mFn:%s\x1b[0m> ", "register", string(respMessage.Data[1])))
	switch string(respMessage.Data[1]) {
	case "E":
		return fmt.Errorf("connection refused. register error: %q", respMessage)
	case "R":
		if len(data) < 5 {
			return fmt.Errorf("connection refused. register response incorrect: %x", data)
		}
		chain := mode.ChainName()
		respChain := string(respMessage.Data[0])
		if respChain != chain {
			return fmt.Errorf("connection refused. Expected chain: %q but received: %q", chain, respChain)
		}
		var peerListener Addrs
		respID, errID := peer.IDFromBytes(respMessage.Data[2])
		errAddr := proto.Unmarshal(respMessage.Data[2], &peerListener)
		if errID != nil || errAddr != nil {
			return errors.New("Unmarshal error")
		}
		timestamp := binary.BigEndian.Uint64(respMessage.Data[3])
		log.Infof("connection established. register replied: Chain:%s PeerID: %x:  listeners: %v  timestamp: %d",
			string(respMessage.Data[0]), respID, util.GetMultiAddrsFromBytes(peerListener.Address), timestamp)
		messagebus.Bus.Announce.Send(string(respMessage.Data[1]), respMessage.Data[2], respMessage.Data[3], respMessage.Data[4])
		//client.Send(fn, chain, n.publicKey, n.listeners, timestamp)
	}
	return nil
}

func (n *Node) register(peerInfo *peerlib.AddrInfo) (*network.Stream, error) {
	s, err := n.Host.NewStream(context.Background(), peerInfo.ID, "p2pstream")
	defer s.Close()
	n.log.Info("---- start to register ---")
	if err != nil {
		n.log.Warn(err.Error())
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	p2pPacked := n.packP2PMessage()
	if nil == p2pPacked {
		n.log.Error("packed message error")
		return nil, err
	}
	_, err = rw.Write(append(p2pPacked, '\r', '\n'))
	if err != nil {
		n.log.Error(err.Error())
		return nil, err
	}
	rw.Flush()
	// Wait for response
	resp := make([]byte, maxBytesRecieve)
	lenByte, err := rw.Read(resp)
	log.Info(fmt.Sprintf("%s RECIEVE:\x1b[32m%d\x1b[0m> ", "listener", lenByte))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	if lenByte < 1 {
		log.Error("length of byte recieved is less than 1")
	}
	respMessageUnPacked := P2PMessage{}
	proto.Unmarshal(resp, &respMessageUnPacked)
	if len(respMessageUnPacked.Data) == 0 {
		return nil, errors.New("unmarshal response error")
	}
	//printP2PMessage(respMessageUnPacked, log)
	reqID, err := peerlib.IDFromBytes(respMessageUnPacked.Data[2])
	if err != nil {
		log.Infof(err.Error())
		return nil, err
	}
	chain := string(respMessageUnPacked.Data[0])
	fn := string(respMessageUnPacked.Data[1])
	log.Infof("chain:%s, fn:%s reqID:%s", chain, fn, reqID.String())
	var respListener Addrs
	errAddr := proto.Unmarshal(respMessageUnPacked.Data[3], &respListener)

	if errAddr != nil {
		log.Error(errAddr.Error())
		return nil, errAddr
	}
	respMaAddrs := util.GetMultiAddrsFromBytes(respListener.Address)
	timestamp := binary.BigEndian.Uint64(respMessageUnPacked.Data[4])
	log.Infof("chain:%s, fn:%s reqID:%s timestamp%d listeners:%s", chain, fn, reqID.String(), timestamp, util.PrintMaAddrs(respMaAddrs))
	announce.AddPeer(reqID, respMaAddrs, timestamp) // id, listeners, timestam
	return &s, nil
}

func (n *Node) packP2PMessage() []byte {
	chain := "local"
	fn := "R"
	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	addrsPacked, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(n.Announce)})
	if err != nil {
		n.log.Info("addrsPacked marshal error")
		return nil
	}
	id, idErr := n.Host.ID().Marshal()
	if idErr != nil {
		n.log.Info("idErr marshal error")
		return nil
	}
	reqData := [][]byte{[]byte(chain), []byte(fn), id, addrsPacked, timestamp}
	reqMsgPacked, err := proto.Marshal(&P2PMessage{Data: reqData})
	if err != nil {
		n.log.Info("packed message marshal error")
		return nil
	}
	return reqMsgPacked
}
