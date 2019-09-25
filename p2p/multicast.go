package p2p

import (
	"bitmark-network/messagebus"
	"context"
	"fmt"
	"os"

	"github.com/bitmark-inc/bitmarkd/mode"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// SubHandler multicasting subscription handler
func (n *Node) SubHandler(ctx context.Context, sub *pubsub.Subscription) {
	log := n.log
	log.Info("-- Sub start listen --")

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		//TODO: Check if it works after some modification on pb file
		req := &P2PMessage{}
		err = proto.Unmarshal(msg.Data, req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		reqChain := string(req.Data[0])
		fn := string(req.Data[1])
		parameters := req.Data[2:]

		switch fn {
		case "peer":
			if reqChain != mode.ChainName() {
				break
			}
			dataLength := len(parameters)
			if dataLength < 3 {
				log.Debugf("peer with too few data: %d items", dataLength)
				break
			}

			if 8 != len(parameters[2]) {
				log.Debugf("peer with invalid timestamp=%v", parameters[2])
				break
			}
			id, err := peer.IDFromBytes(parameters[0])
			log.Infof("-->>sub Recieve: %v  ID:%s \n", fn, id.ShortString())
			if err != nil {
				log.Error("invalid id in requesting")
			}
			messagebus.Bus.Announce.Send("addpeer", parameters[0], parameters[1], parameters[2])
		default:
			log.Infof("unreganized Command:%s ", 0)
		}
	}
}
