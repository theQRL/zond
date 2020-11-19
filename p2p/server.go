package p2p

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/p2p/messages"
	"github.com/theQRL/zond/protos"
	"github.com/willf/bloom"
	"net"
	"strconv"
	"sync"
	"time"
)

type conn struct {
	fd      net.Conn
	inbound bool
}

type peerDrop struct {
	*Peer
	err       error
	requested bool  // true if signaled by the peer
}

type PeerInfo struct {
	IP                      string `json:"IP"`
	Port                    uint16 `json:"Port"`
	LastConnectionTimestamp uint64 `json:"LastConnectionTimestamp"`
}

type PeersInfo struct {
	PeersInfo []PeerInfo `json:"PeersInfo"`
}

type PeerIPWithPLData struct {
	IP     string
	PLData *protos.PLData
}

type Server struct {
	config *config.Config

	chain        *chain.Chain
	ntp          ntp.NTPInterface
	//peersInfo    *PeersInfo
	peerData     *metadata.PeerData
	ipCount      map[string]int
	inboundCount uint16

	listener     net.Listener
	lock         sync.Mutex
	peerInfoLock sync.Mutex

	running bool
	loopWG  sync.WaitGroup

	exit                        chan struct{}
	connectPeersExit            chan struct{}
	mrDataConn                  chan *MRDataConn
	addPeerToPeerList           chan *PeerIPWithPLData
	blockAndPeerChan            chan *BlockAndPeer
	addPeer                     chan *conn
	delPeer                     chan *peerDrop
	registerAndBroadcastChan    chan *messages.RegisterMessage
	blockReceivedForAttestation chan *block.Block
	attestationReceivedForBlock chan *transactions.Attest

	filter          *bloom.BloomFilter
	mr              *MessageReceipt
	downloader		*Downloader
	messagePriority map[protos.LegacyMessage_FuncName]uint64
}

func (srv *Server) GetRegisterAndBroadcastChan() chan *messages.RegisterMessage {
	return srv.registerAndBroadcastChan
}

func (srv *Server) GetBlockReceivedForAttestation() chan *block.Block {
	return srv.blockReceivedForAttestation
}

func (srv *Server) GetAttestationReceivedForBlock() chan *transactions.Attest {
	return srv.attestationReceivedForBlock
}

func (srv *Server) BroadcastBlock(block *block.Block) {
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_BK,
			Data: &protos.LegacyMessage_Block {
				Block: block.PBData(),
			},
		},
		MsgHash: misc.Bin2HStr(block.HeaderHash()),
	}
	srv.registerAndBroadcastChan <-msg
}

func (srv *Server) BroadcastBlockForAttestation(block *block.Block, signature []byte) {
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_BA,
			Data: &protos.LegacyMessage_BlockForAttestation{
				BlockForAttestation: &protos.BlockForAttestation{
					Block: block.PBData(),
					Signature: signature,
				},
			},
		},
		MsgHash: misc.Bin2HStr(block.PartialBlockSigningHash()),
	}
	srv.registerAndBroadcastChan <-msg
}

func (srv *Server) BroadcastAttestationTransaction(attestTx *transactions.Attest,
	slotNumber uint64, blockProposer []byte,
	parentHeaderHash []byte, partialBlockSigningHash []byte) {
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_AT,
			Data: &protos.LegacyMessage_AtData{
				AtData: &protos.ProtocolTransactionData{
					Tx: attestTx.PBData(),
					SlotNumber: slotNumber,
					BlockProposer: blockProposer,
					ParentHeaderHash: parentHeaderHash,
					PartialBlockSigningHash: partialBlockSigningHash,
				},
			},
		},
		MsgHash: misc.Bin2HStr(attestTx.TxHash(attestTx.GetSigningHash(partialBlockSigningHash))),
	}
	log.Info("[BroadcastAttestationTransaction] Broadcasting Attestation Txn ",
		msg.MsgHash)
	srv.registerAndBroadcastChan <-msg
}

func (srv *Server) Start() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server is already running")
	}

	srv.filter = bloom.New(200000, 5)
	//srv.chain.GetTransactionPool().SetRegisterAndBroadcastChan(srv.registerAndBroadcastChan)
	if err := srv.startListening(); err != nil {
		return err
	}

	srv.running = true
	go srv.run()
	go srv.downloader.DownloadMonitor()
	go srv.ConnectPeers()

	return nil
}

func (srv *Server) Stop() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if !srv.running {
		return
	}
	srv.running = false
	if srv.listener != nil {
		srv.listener.Close()
	}

	close(srv.exit)
	close(srv.connectPeersExit)
	srv.loopWG.Wait()

	return nil
}

func (srv *Server) listenLoop(listener net.Listener) {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	for {
		c, err := listener.Accept()

		if err != nil {
			log.Error("Read ERROR Reason: ", err)
			return
		}
		log.Info("New Peer joined")
		srv.addPeer <- &conn{c, true}
	}
}

func (srv *Server) ConnectPeer(peer string) error {
	ip, _, _ := net.SplitHostPort(peer)
	if _, ok := srv.ipCount[ip]; ok {
		return nil
	}

	c, err := net.DialTimeout("tcp", peer, 10 * time.Second)

	if err != nil {
		log.Warn("Error while connecting to Peer ", peer)
		return err
	}
	log.Info("Connected to peer ", peer)
	srv.addPeer <- &conn{c, false}

	return nil
}

func (srv *Server) ConnectPeers() error {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	for _, peer := range srv.config.User.Node.PeerList {
		log.Info("Connecting peer ", peer)
		srv.ConnectPeer(peer)
		// TODO: Update last connection time
	}

	for {
		select {
		case <-time.After(15*time.Second):
			srv.peerInfoLock.Lock()
			if srv.inboundCount > srv.config.User.Node.MaxPeersLimit {
				srv.peerInfoLock.Unlock()
				break
			}

			maxConnectionTry := 10
			peerList := make([]string, 0)

			count := 0

			for _, p := range srv.peerData.DisconnectedPeers() {

				if count > maxConnectionTry {
					break
				}

				if _, ok := srv.ipCount[p.IP()]; ok {
					continue
				}

				count += 1
				peerList = append(peerList, p.IPPort())
				//p.LastConnectionTimestamp = srv.ntp.Time()
			}
			srv.peerInfoLock.Unlock()

			for _, ipPort := range peerList {
				if !srv.running {
					break
				}
				fmt.Println("Trying to Connect",
					"Peer", ipPort)
				err := srv.ConnectPeer(ipPort)
				if err != nil {
					log.Info("Failed to connect to ", ipPort)
				}
			}

		case <-srv.connectPeersExit:
			return nil
		}
	}
}

func (srv *Server) startListening() error {
	bindingAddress := fmt.Sprintf("%s:%d",
		srv.config.User.Node.BindingIP,
		srv.config.User.Node.LocalPort)

	listener, err := net.Listen("tcp", bindingAddress)
	if err != nil {
		return err
	}

	srv.listener = listener
	go srv.listenLoop(listener)

	return nil
}

func (srv *Server) run() {
	var (
		peers        = make(map[string]*Peer)
	)

	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

running:
	for {
		select {
		case <-srv.exit:
			srv.downloader.Exit()
			log.Debug("Shutting Down Server")
			break running
		case c := <-srv.addPeer:
			srv.peerInfoLock.Lock()

			log.Debug("Adding peer",
				" addr ", c.fd.RemoteAddr())
			p := newPeer(
				&c.fd,
				c.inbound,
				srv.chain,
				srv.filter,
				srv.mr,
				srv.mrDataConn,
				srv.registerAndBroadcastChan,
				srv.blockReceivedForAttestation,
				srv.attestationReceivedForBlock,
				srv.addPeerToPeerList,
				srv.blockAndPeerChan,
				srv.messagePriority)
			go srv.runPeer(p)
			peers[c.fd.RemoteAddr().String()] = p

			ip, _, _ := net.SplitHostPort(c.fd.RemoteAddr().String())
			srv.ipCount[ip] += 1
			if p.inbound {
				srv.inboundCount++
			}
			
			if srv.ipCount[ip] > srv.config.User.Node.MaxRedundantConnections {
				p.Disconnect()
				// TODO: Ban peer
			}

			srv.peerInfoLock.Unlock()
			srv.downloader.AddPeer(p)

		case pd := <-srv.delPeer:
			srv.peerInfoLock.Lock()

			log.Debug("Removing Peer", "err", pd.err)
			peer := peers[pd.conn.RemoteAddr().String()]
			delete(peers, pd.conn.RemoteAddr().String())
			if pd.inbound {
				srv.inboundCount--
			}
			ip, _, _ := net.SplitHostPort(pd.conn.RemoteAddr().String())
			srv.ipCount[ip] -= 1
			srv.peerInfoLock.Unlock()

			srv.downloader.RemovePeer(peer)

		case mrDataConn := <-srv.mrDataConn:
			// TODO: Process Message Receipt
			// Need to get connection too
			mrData := mrDataConn.mrData
			msgHash := misc.Bin2HStr(mrData.Hash)
			switch mrData.Type {
			case protos.LegacyMessage_BA:
				/*
				1. Verify if Block Received for attestation is valid
				2. Broadcast the block
				3. Attest the block if Staking is Enabled on this node
				 */
				_, err := srv.chain.GetBlock(mrData.ParentHeaderHash)
				if err != nil {
					log.Info("[BlockForAttestation] Missing Parent Block",
						" #", mrData.SlotNumber,
						" Partial Block Signing Hash ", misc.Bin2HStr(mrData.Hash),
						" Parent Block ", misc.Bin2HStr(mrData.ParentHeaderHash))
					break
				}

				if srv.mr.contains(mrData.Hash, mrData.Type) {
					break
				}

				srv.mr.addPeer(mrData, mrDataConn.peer)

				value, ok := srv.mr.GetRequestedHash(msgHash)
				if ok && value.GetRequested() {
					break
				}

				go srv.RequestFullMessage(mrData)

				//TODO: Logic to be written
				//srv.blockReceivedForAttestation
			case protos.LegacyMessage_BK:
				finalizedHeaderHash, err := srv.chain.GetFinalizedHeaderHash()
				if err != nil {
					log.Error("No Finalized Header Hash ", err)
				}
				if finalizedHeaderHash != nil {
					finalizedBlock, err := srv.chain.GetBlock(finalizedHeaderHash)
					if err != nil {
						log.Error("Failed to get finalized block ",
							misc.Bin2HStr(finalizedHeaderHash))
						break
					}
					// skip slot number beyond the Finalized slot Number
					if finalizedBlock.SlotNumber() >= mrData.SlotNumber {
						log.Warn("[BlockReceived] Block #", mrData.SlotNumber,
							" is beyond finalized block #", finalizedBlock.SlotNumber())
						break
					}
				}

				_, err = srv.chain.GetBlock(mrData.ParentHeaderHash)
				if err != nil {
					log.Info("[BlockReceived] Missing Parent Block ",
						" #", mrData.SlotNumber,
						" Block ", misc.Bin2HStr(mrData.Hash),
						" Parent Block ", misc.Bin2HStr(mrData.ParentHeaderHash))
					break
				}

				if srv.mr.contains(mrData.Hash, mrData.Type) {
					break
				}

				srv.mr.addPeer(mrData, mrDataConn.peer)

				value, ok := srv.mr.GetRequestedHash(msgHash)
				if ok && value.GetRequested() {
					break
				}

				go srv.RequestFullMessage(mrData)
				// Request for full message
				// Check if its already being feeded by any other peer
			case protos.LegacyMessage_TT:
				srv.HandleTransaction(mrDataConn)
			case protos.LegacyMessage_ST:
				srv.HandleTransaction(mrDataConn)
			case protos.LegacyMessage_AT:
				srv.HandleTransaction(mrDataConn)
			default:
				log.Warn("Unknown Message Receipt Type",
					"Type", mrData.Type)
				mrDataConn.peer.Disconnect()
			}
		case blockAndPeer := <-srv.blockAndPeerChan:
			srv.BlockReceived(blockAndPeer.peer, blockAndPeer.block)
		case addPeerToPeerList := <-srv.addPeerToPeerList:
			srv.UpdatePeerList(addPeerToPeerList)
		case registerAndBroadcast := <-srv.registerAndBroadcastChan:
			srv.mr.Register(registerAndBroadcast.MsgHash, registerAndBroadcast.Msg)

			out := &Msg{
				msg: &protos.LegacyMessage {
					FuncName: protos.LegacyMessage_MR,
					Data: &protos.LegacyMessage_MrData {
						MrData: &protos.MRData {
							Hash: misc.HStr2Bin(registerAndBroadcast.MsgHash),
							Type: registerAndBroadcast.Msg.FuncName,
						},
					},
				},
			}
			b := registerAndBroadcast.Msg.GetBlock()
			if b != nil {
				out.msg.GetMrData().SlotNumber = b.Header.SlotNumber
				out.msg.GetMrData().ParentHeaderHash = b.Header.ParentHeaderHash
			} else {
				ba := registerAndBroadcast.Msg.GetBlockForAttestation()
				if ba != nil {
					out.msg.GetMrData().SlotNumber = ba.Block.Header.SlotNumber
					out.msg.GetMrData().ParentHeaderHash = ba.Block.Header.ParentHeaderHash
				}
			}
			ignorePeers := make(map[*Peer]bool, 0)
			if msgRequest, ok := srv.mr.GetRequestedHash(registerAndBroadcast.MsgHash); ok {
				ignorePeers = msgRequest.peers
			}
			for _, p := range peers {
				if _, ok := ignorePeers[p]; !ok {
					p.Send(out)
				}
			}
		}
	}
	for _, p := range peers {
		p.Disconnect()
	}
}

func (srv *Server) HandleTransaction(mrDataConn *MRDataConn) {
	mrData := mrDataConn.mrData
	srv.mr.addPeer(mrData, mrDataConn.peer)

	// TODO: Ignore transaction if node is syncing
	if srv.downloader.isSyncing {
		return
	}
	if srv.chain.GetTransactionPool().IsFull() {
		return
	}
	go srv.RequestFullMessage(mrData)
}

func (srv *Server) RequestFullMessage(mrData *protos.MRData) {
	for {
		msgHash := misc.Bin2HStr(mrData.Hash)
		_, ok := srv.mr.GetHashMsg(msgHash)
		if ok {
			if _, ok = srv.mr.GetRequestedHash(msgHash); ok {
				srv.mr.RemoveRequestedHash(msgHash)
			}
			return
		}
		requestedHash, ok := srv.mr.GetRequestedHash(msgHash)
		if !ok {
			return
		}
		peer := requestedHash.GetPeer()
		if peer == nil {
			return
		}
		requestedHash.SetPeer(peer, true)
		mrData := &protos.MRData{
			Hash: mrData.Hash,
			Type: mrData.Type,
		}
		out := &Msg{}
		out.msg = &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_SFM,
			Data: &protos.LegacyMessage_MrData {
				MrData: mrData,
			},
		}
		peer.Send(out)

		time.Sleep(time.Duration(srv.config.Dev.MessageReceiptTimeout) * time.Second)
	}
}

func (srv *Server) BlockReceived(peer *Peer, b *block.Block) {
	headerHash := misc.Bin2HStr(b.HeaderHash())
	log.Info(">>> Received Block",
		" #", b.SlotNumber(),
		" HeaderHash ", headerHash)

	// TODO: Trigger Syncing/Block downloader
	select {
	case srv.downloader.blockAndPeerChannel <- &BlockAndPeer{b, peer}:
	case <-time.After(5 * time.Second):
		log.Info("Timeout for Received Block",
			"#", b.SlotNumber(),
			"HeaderHash", headerHash)
	}
}

func (srv *Server) UpdatePeerList(p *PeerIPWithPLData) error {
	peerIP := p.IP
	peerPort := p.PLData.PublicPort
	if !(peerPort > 0 && peerPort < 65536) {
		log.Warn("Invalid PublicPort ", peerPort, " shared by ", peerIP)
		return nil
	}
	err := srv.peerData.AddConnectedPeers(peerIP, strconv.FormatUint(uint64(peerPort), 10))
	if err != nil {
		log.Error("Failed to Add Peer into peer list",
			" ", peerIP, ":", peerPort,
			" Reason: ", err.Error())
		return err
	}
	for _, peerIPPort := range p.PLData.PeerIps {
		peerIP, peerPort, err := net.SplitHostPort(peerIPPort)
		if err != nil {
			log.Error("Failed to SplitHostPort ", peerIPPort,
				" provided by ", p.IP)
			continue
		}
		intPeerPort, err := strconv.ParseUint(peerPort, 10, 32)
		if err != nil {
			log.Error("Failed to parse PeerPort ", peerPort,
				" shared by ", p.IP,
				" Reason: ", err.Error())
			return err
		}
		if !(intPeerPort > 0 && intPeerPort < 65536) {
			log.Warn("Invalid PublicPort ", intPeerPort, " shared by ", p.IP)
			return nil
		}
		if srv.peerData.IsPeerInList(peerIP, peerPort) {
			continue
		}
		err = srv.peerData.AddDisconnectedPeers(peerIP, peerPort)
		if err != nil {
			log.Error("Failed to add peer ", peerIP, ":", peerPort, " in peer list",
				" Reason: ", err.Error())
			continue
		}
	}
	return nil
}

func (srv *Server) runPeer(p *Peer) {
	remoteRequested := p.run()

	srv.delPeer <- &peerDrop{p, nil, remoteRequested}
}

func NewServer(chain *chain.Chain) (*Server, error) {
	peerData, err := metadata.NewPeerData()
	if err != nil {
		return nil, err
	}
	srv := &Server{
		config: config.GetConfig(),
		chain: chain,
		ntp: ntp.GetNTP(),
		peerData: peerData,
		ipCount: make(map[string]int),
		mr: CreateMR(),
		downloader: NewDownloader(chain),

		exit: make(chan struct{}),
		connectPeersExit: make(chan struct{}),
		mrDataConn: make(chan *MRDataConn),
		addPeerToPeerList: make(chan *PeerIPWithPLData),
		blockAndPeerChan: make(chan *BlockAndPeer),
		addPeer: make(chan *conn),
		delPeer: make(chan *peerDrop),
		registerAndBroadcastChan: make(chan *messages.RegisterMessage, 100),
		blockReceivedForAttestation: make(chan *block.Block),
		attestationReceivedForBlock: make(chan *transactions.Attest),

		messagePriority: make(map[protos.LegacyMessage_FuncName]uint64),
	}

	srv.messagePriority[protos.LegacyMessage_VE] = 0
	srv.messagePriority[protos.LegacyMessage_PL] = 0
	srv.messagePriority[protos.LegacyMessage_PONG] = 0

	srv.messagePriority[protos.LegacyMessage_MR] = 2
	srv.messagePriority[protos.LegacyMessage_SFM] = 1

	srv.messagePriority[protos.LegacyMessage_BA] = 1
	srv.messagePriority[protos.LegacyMessage_BK] = 1
	srv.messagePriority[protos.LegacyMessage_FB] = 0
	srv.messagePriority[protos.LegacyMessage_PB] = 0

	srv.messagePriority[protos.LegacyMessage_TT] = 1
	srv.messagePriority[protos.LegacyMessage_ST] = 1
	srv.messagePriority[protos.LegacyMessage_AT] = 1

	srv.messagePriority[protos.LegacyMessage_SYNC] = 0
	srv.messagePriority[protos.LegacyMessage_CHAINSTATE] = 0
	srv.messagePriority[protos.LegacyMessage_EBHREQ] = 0
	srv.messagePriority[protos.LegacyMessage_EBHRESP] = 0
	srv.messagePriority[protos.LegacyMessage_P2P_ACK] = 0

	return srv, nil
}
