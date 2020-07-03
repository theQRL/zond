package p2p

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/proto/p2p"
	"io"
)

type PeerClient struct {
	peer p2p.P2PProtocol_TransmitServer

	exit   chan struct{}
}

func (p *PeerClient) Send(p2pMessage *p2p.P2PMessage) error {
	err := p.peer.Send(p2pMessage)
	if err != nil {
		log.Error("Error while transmitting message to client ", err)
	}
	return err
}

func (p *PeerClient) Recv() {
	for {
		msg, err := p.peer.Recv()
		if err == io.EOF {
			log.Info("Client Disconnected")
			return
		}
		if err != nil {
			log.Warn("Received Error %v", err)
			continue
		}
		switch msg.Type.(type) {
		case *p2p.P2PMessage_Req:
			//p.HandleRequest(srv, msg.GetReq())
		case *p2p.P2PMessage_Resp:
			//p.HandleResponse(srv, msg.GetResp())
		}
	}
}

func NewPeerClient(peer p2p.P2PProtocol_TransmitServer) *PeerClient {
	return &PeerClient {
		peer: peer,
		exit: make(chan struct{}),
	}
}
