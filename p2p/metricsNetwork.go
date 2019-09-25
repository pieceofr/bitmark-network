package p2p

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/logger"
	p2pcore "github.com/libp2p/go-libp2p-core"
	p2pnet "github.com/libp2p/go-libp2p-core/network"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// MetricsNetwork contain P2P network metrics
type MetricsNetwork struct {
	log         *logger.L
	streamCount counter.Counter
	connCount   counter.Counter
}

//NewMetricsNetwork return a metricNetwork
func NewMetricsNetwork(l *logger.L) MetricsNetwork {
	return MetricsNetwork{log: l}
}

func (m *MetricsNetwork) startMonitor(host p2pcore.Host) {
	log := m.log
	host.Network().Notify(&p2pnet.NotifyBundle{
		ListenF: func(net p2pnet.Network, addr multiaddr.Multiaddr) {
			log.Debugf("@@Host: %v is listen at %v\n", addr.String(), time.Now())
		},
		ConnectedF: func(net p2pnet.Network, conn p2pnet.Conn) {
			m.connCount.Increment()
			log.Infof("@@: Conn: %v Connected at %v ConnCount:%d\n", conn.RemoteMultiaddr().String(), time.Now(), m.connCount)
		},
		DisconnectedF: func(net p2pnet.Network, conn p2pnet.Conn) {
			m.connCount.Decrement()
			log.Infof("@@Conn: %v Disconnected at %v  ConnCount:%d\n", conn.RemoteMultiaddr().String(), time.Now(), m.connCount)
		},
		OpenedStreamF: func(net p2pnet.Network, stream p2pnet.Stream) {
			m.streamCount.Increment()
			log.Debugf("@@Stream : %v-%v is Opened at %v streamCount:%d\n", stream.Conn().RemoteMultiaddr().String(), stream.Protocol(), time.Now(), m.streamCount)
		},
		ClosedStreamF: func(net p2pnet.Network, stream p2pnet.Stream) {
			m.streamCount.Decrement()
			log.Debugf("@@Stream :%v-%v is Closed at %v streamCount:%d\n", stream.Conn().RemoteMultiaddr().String(), stream.Protocol(), time.Now(), m.streamCount)
		},
	})
}
