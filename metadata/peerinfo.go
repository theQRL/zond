package metadata

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/protos"
	"strings"
)

type PeerInfo struct {
	pbData *protos.PeerInfo
}

func (p *PeerInfo) IP() string {
	return p.pbData.Ip
}

func (p *PeerInfo) MultiAddr() string {
	return p.pbData.MultiAddr
}

func (p *PeerInfo) Identity() string {
	return p.pbData.Identity
}

func (p *PeerInfo) Timestamp() uint64 {
	return p.pbData.Timestamp
}

func (p *PeerInfo) IsSame(p1 *PeerInfo) bool {
	return p.Identity() == p1.Identity()
}

func NewPeerInfo(multiAddr string,
	timestamp uint64) *PeerInfo {
	info := strings.Split(multiAddr, "/")
	if len(info) != 7 {
		log.Error("[PeerInfo] Invalid multiAddr")
		return nil
	}
	identity := info[6]
	ip := info[2]
	return &PeerInfo{
		pbData: &protos.PeerInfo{
			Ip: ip,
			MultiAddr: multiAddr,
			Identity: identity,
			Timestamp: timestamp,
		},
	}
}
