package metadata

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

type PeerData struct {
	pbData *protos.PeerData

	connectedPeers    []*PeerInfo
	disconnectedPeers []*PeerInfo

	lock sync.Mutex
}

func (p *PeerData) ConnectedPeers() []*PeerInfo {
	return p.connectedPeers
}

func (p *PeerData) DisconnectedPeers() []*PeerInfo {
	return p.disconnectedPeers
}

func (p *PeerData) findIndex(peerInfo *PeerInfo,
	peerList []*PeerInfo) int {
	for i, pI := range peerList {
		if peerInfo.IsSame(pI) {
			return i
		}
	}
	return -1
}

func (p *PeerData) PeerList() []string {
	p.lock.Lock()
	defer p.lock.Unlock()

	var peerList []string

	for _, peerInfo := range p.connectedPeers {
		peerList = append(peerList, peerInfo.IPPort())
	}

	for _, peerInfo := range p.disconnectedPeers {
		peerList = append(peerList, peerInfo.IPPort())
	}

	return peerList
}

func (p *PeerData) IsPeerInList(ip string, port string) bool {
	/*
	Check if Peer IP and Port is already added either
	into connected peer list or disconnected peer list
	 */
	p.lock.Lock()
	defer p.lock.Unlock()
	peerInfo := NewPeerInfo(ip, port, 0)
	index := p.findIndex(peerInfo, p.connectedPeers)
	if index != -1 {
		return true
	}
	index = p.findIndex(peerInfo, p.disconnectedPeers)
	if index != -1 {
		return true
	}
	return false
}

func (p *PeerData) removeConnectedPeers(peerInfo *PeerInfo) {
	index := p.findIndex(peerInfo, p.connectedPeers)
	if index != -1 {
		return
	}
	p.connectedPeers = append(p.connectedPeers[:index],
		p.connectedPeers[index+1:]...)
}

func (p *PeerData) removeDisconnectedPeers(peerInfo *PeerInfo) {
	index := p.findIndex(peerInfo, p.disconnectedPeers)
	if index != -1 {
		return
	}
	p.disconnectedPeers = append(p.disconnectedPeers[:index],
		p.disconnectedPeers[index+1:]...)
}

func (p *PeerData) AddConnectedPeers(ip string, port string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	timestamp := ntp.GetNTP().Time()
	peerInfo := NewPeerInfo(ip, port, timestamp)

	index := p.findIndex(peerInfo, p.connectedPeers)
	if index == -1 {
		p.connectedPeers = append(p.connectedPeers, peerInfo)
	}

	p.removeDisconnectedPeers(peerInfo)

	if err := p.Save(); err != nil {
		return err
	}
	return nil
}

func (p *PeerData) AddDisconnectedPeers(ip string, port string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	timestamp := ntp.GetNTP().Time()
	peerInfo := NewPeerInfo(ip, port, timestamp)

	index := p.findIndex(peerInfo, p.disconnectedPeers)
	if index == -1 {
		p.disconnectedPeers = append([]*PeerInfo{peerInfo},
		p.disconnectedPeers...)
	}

	p.removeConnectedPeers(peerInfo)

	if err := p.Save(); err != nil {
		return err
	}
	return nil
}

func (p *PeerData) RemovePeer(ip string, port string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	timestamp := ntp.GetNTP().Time()
	peerInfo := NewPeerInfo(ip, port, timestamp)

	p.removeConnectedPeers(peerInfo)
	p.removeDisconnectedPeers(peerInfo)

	if err := p.Save(); err != nil {
		return err
	}
	return nil
}

func (p *PeerData) Save() error {
	p.pbData = &protos.PeerData{}
	for _, peerInfo := range p.connectedPeers {
		p.pbData.ConnectedPeers = append(p.pbData.ConnectedPeers,
			peerInfo.pbData)
	}

	for _, peerInfo := range p.disconnectedPeers {
		p.pbData.DisconnectedPeers = append(p.pbData.DisconnectedPeers,
			peerInfo.pbData)
	}

	jsonData, err := protojson.Marshal(p.pbData)
	if err != nil {
		log.Error("Error serializing PeerData into json data ",
			err.Error())
		return err
	}

	fileName := path.Join(config.GetUserConfig().DataDir(),
		config.GetDevConfig().PeersFilename)

	if err := ioutil.WriteFile(fileName, jsonData, 0644); err != nil {
		log.Error("Error writing peers file ", err.Error())
		return err
	}

	return nil
}

func (p *PeerData) Load() error {
	fileName := path.Join(config.GetUserConfig().DataDir(),
		config.GetDevConfig().PeersFilename)
	f, err := os.Open(fileName)
	if err != nil {
		// Create new file, if unable to open
		return p.Save()
	}
	defer f.Close()
	jsonData := make([]byte, 0)
	jsonData, err = ioutil.ReadAll(f)
	if err != nil {
		log.Error("Error while reading file ", err.Error())
		return err
	}

	err = json.Unmarshal(jsonData, p.pbData)
	if err != nil {
		log.Error("Error deserializing PeerData ", err.Error())
		return err
	}

	p.pbData.DisconnectedPeers = append(p.pbData.ConnectedPeers,
		p.pbData.DisconnectedPeers...)
	p.pbData.ConnectedPeers = make([]*protos.PeerInfo, 0)

	for _, peerInfoPBData := range p.pbData.DisconnectedPeers {
		p1 := NewPeerInfo(peerInfoPBData.Ip, peerInfoPBData.Port, peerInfoPBData.Timestamp)
		p.disconnectedPeers = append(p.disconnectedPeers, p1)
	}

	return nil
}

func NewPeerData() (*PeerData, error) {
	p := &PeerData{
		connectedPeers: make([]*PeerInfo, 0),
		disconnectedPeers: make([]*PeerInfo, 0),
	}
	err := p.Load()
	if err != nil {
		log.Error("Failed to load PeerData ", err.Error())
		return nil, err
	}
	return &PeerData {
		pbData: &protos.PeerData{
			ConnectedPeers: make([]*protos.PeerInfo, 0),
			DisconnectedPeers: make([]*protos.PeerInfo, 0),
		},
	}, nil
}
