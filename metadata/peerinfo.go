package metadata

import (
	"fmt"
	"github.com/theQRL/zond/protos"
)

type PeerInfo struct {
	pbData *protos.PeerInfo
}

func (p *PeerInfo) IP() string {
	return p.pbData.Ip
}

func (p *PeerInfo) Port() uint32 {
	return p.pbData.Port
}

func (p *PeerInfo) Timestamp() uint64 {
	return p.pbData.Timestamp
}

func (p *PeerInfo) IPPort() string {
	return fmt.Sprintf("%s:%d", p.IP(), p.Port())
}

func (p *PeerInfo) IsSame(p1 *PeerInfo) bool {
	return p.IP() == p1.IP() && p.Port() == p1.Port()
}

func NewPeerInfo(ip string, port uint32, timestamp uint64) *PeerInfo {
	return &PeerInfo{
		pbData: &protos.PeerInfo{
			Ip: ip,
			Port: port,
			Timestamp: timestamp,
		},
	}
}
