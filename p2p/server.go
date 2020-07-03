package p2p

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/proto/p2p"
	"google.golang.org/grpc"
	"io"
	"net"
	"sync"
)

type P2PServer struct {
	running  bool
	exit     chan struct{}
	chanSend chan *p2p.P2PMessage

	lock   sync.Mutex
	loopWG sync.WaitGroup

	grpcServer *grpc.Server

	p2p.UnimplementedP2PProtocolServer
}

func (p *P2PServer) HandleRequest(srv p2p.P2PProtocol_TransmitServer, req *p2p.Request) {
	resp := &p2p.Response{}

	switch req.Type.(type) {
	case *p2p.Request_PingReq:
		pingResp := &p2p.PingResp {
			Timestamp: 1000,
		}
		resp.Type = &p2p.Response_PingResp {
			PingResp: pingResp,
		}

	case *p2p.Request_BlockReq:
		blockResp := &p2p.BlockResp {
			BlockNumber: 9898,
		}
		resp.Type = &p2p.Response_BlockResp {
			BlockResp: blockResp,
		}
	}

	msgResp := &p2p.P2PMessage{
		Type: &p2p.P2PMessage_Resp {
			Resp: resp,
		},
	}
	log.Info("Request from Client", req.GetBlockReq().BlockNumber)
	if err := srv.Send(msgResp); err != nil {
		log.Error("Error while transmitting message to client ", err)
	}
}

func (p *P2PServer) HandleResponse(srv p2p.P2PProtocol_TransmitServer, resp *p2p.Response) {
	switch resp.Type.(type) {
	case *p2p.Response_PingResp:
		resp.GetPingResp()

	case *p2p.Response_BlockResp:
		resp.GetBlockResp()
	}
}

func (p *P2PServer) Send(srv p2p.P2PProtocol_TransmitServer) {

}

func (p *P2PServer) Transmit(srv p2p.P2PProtocol_TransmitServer) error {
	peer := NewPeerClient(srv)

	ctx := srv.Context()
	var err error

	Loop:
	for {
		select {
		case <- ctx.Done():
			err = ctx.Err()
			break Loop
		case <- p.exit:
			break Loop
		}
	}
	return err
}

func (p *P2PServer) Start() {
	port := 15005
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed %v", err)
		return
	}
	p.grpcServer = grpc.NewServer()

	p2p.RegisterP2PProtocolServer(p.grpcServer, p)
	p.running = true  // TODO: Changing value without lock

	if err = p.grpcServer.Serve(lis); err != nil {
		log.Error("Error while starting listener ", err)
		return
	}
}

func (p *P2PServer) Init() {
	p.exit = make(chan struct{})
}

func (p *P2PServer) Stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if !p.running {
		return
	}
	p.running = false

	p.grpcServer.Stop()

	close(p.exit)
	p.loopWG.Wait()
}

func NewServer() *P2PServer {
	srv := &P2PServer{}
	srv.Init()
	return srv
}