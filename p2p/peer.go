package p2p

import "github.com/theQRL/zond/proto/p2p"

type Peer interface {
	Send(p2pMessage *p2p.P2PMessage) error
	Recv()
}

type Peers struct {
	peers map[string]Peer
}

func (p *Peers) AddNewPeer(peer Peer) {

}

// TODO: Wait for all peers to disconnect
