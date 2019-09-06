package p2p

import (
	"bitmark-network/messagebus"
	"context"
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
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
		req := &P2PMessage{}
		err = proto.Unmarshal(msg.Data, req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if len(req.Parameters) == 3 {
			log.Infof("-->>sub Recieve: %v  ID:%s \n", req.Command, shortID(string(req.Parameters[1])))
		}
		switch req.Command {
		case "peer":
			dataLength := len(req.Parameters)
			if dataLength < 3 {
				log.Debugf("peer with too few data: %d items", dataLength)
				break
			}
			if 8 != len(req.Parameters[2]) {
				log.Debugf("peer with invalid timestamp=%v", req.Parameters[2])
				break
			}
			messagebus.Bus.Announce.Send("addpeer", req.Parameters[0], req.Parameters[1], req.Parameters[2])
		}
	}
}
