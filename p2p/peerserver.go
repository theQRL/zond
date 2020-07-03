package p2p

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/proto/p2p"
	"io"
)

type PeerServer struct {
	peer p2p.P2PProtocol_TransmitClient

	recvExit chan struct{}
}

func (p *PeerServer) Send(p2pMessage *p2p.P2PMessage) error {
	err := p.peer.Send(p2pMessage)
	if err != nil {
		log.Error("Error while transmitting message to client ", err)
	}
	return err
}

func (p *PeerServer) Recv() {
	for {
		msg, err := p.peer.Recv()
		if err == io.EOF {
			log.Info("Server Disconnected")
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

func NewPeerServer(peer p2p.P2PProtocol_TransmitClient) *PeerServer {
	return &PeerServer{
		peer: peer,
		recvExit: make(chan struct{}),
	}
}
