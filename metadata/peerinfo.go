package metadata

import (
	"github.com/theQRL/zond/protos"
	"net"
)

type PeerInfo struct {
	pbData *protos.PeerInfo
}

func (p *PeerInfo) IP() string {
	return p.pbData.Ip
}

func (p *PeerInfo) Port() string {
	return p.pbData.Port
}

func (p *PeerInfo) Timestamp() uint64 {
	return p.pbData.Timestamp
}

func (p *PeerInfo) IPPort() string {
	return net.JoinHostPort(p.IP(), p.Port())
}

func (p *PeerInfo) IsSame(p1 *PeerInfo) bool {
	return p.IP() == p1.IP() && p.Port() == p1.Port()
}

func NewPeerInfo(ip string, port string, timestamp uint64) *PeerInfo {
	return &PeerInfo{
		pbData: &protos.PeerInfo{
			Ip: ip,
			Port: port,
			Timestamp: timestamp,
		},
	}
}
