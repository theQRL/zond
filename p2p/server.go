package p2p

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/p2p/messages"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/transactions"
	"github.com/willf/bloom"
	"net"
	"sync"
	"time"
)

type conn struct {
	fd      network.Stream
	inbound bool
}

type peerDrop struct {
	*Peer
	err       error
	requested bool // true if signaled by the peer
}

type PeerIPWithPLData struct {
	multiAddr string
	PLData    *protos.PLData
}

type Server struct {
	config *config.Config

	host             host.Host
	chain            *chain.Chain
	ntp              ntp.NTPInterface
	peerData         *metadata.PeerData
	ipCount          map[string]int
	inboundCount     uint16
	totalConnections uint16

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
	downloader      *Downloader
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
	blockHash := block.Hash()
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_BK,
			Data: &protos.LegacyMessage_Block{
				Block: block.PBData(),
			},
		},
		MsgHash: misc.BytesToHexStr(blockHash[:]),
	}
	srv.registerAndBroadcastChan <- msg
}

func (srv *Server) BroadcastBlockForAttestation(block *block.Block, signature []byte) {
	partialBlockSigningHash := block.PartialBlockSigningHash()
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_BA,
			Data: &protos.LegacyMessage_BlockForAttestation{
				BlockForAttestation: &protos.BlockForAttestation{
					Block:     block.PBData(),
					Signature: signature,
				},
			},
		},
		MsgHash: misc.BytesToHexStr(partialBlockSigningHash[:]),
	}
	srv.registerAndBroadcastChan <- msg
}

func (srv *Server) BroadcastAttestationTransaction(attestTx *transactions.Attest,
	slotNumber uint64, blockProposer []byte,
	parentHeaderHash common.Hash, partialBlockSigningHash common.Hash) {
	txHash := attestTx.TxHash(attestTx.GetSigningHash(partialBlockSigningHash))
	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_AT,
			Data: &protos.LegacyMessage_AtData{
				AtData: &protos.ProtocolTransactionData{
					Tx:                      attestTx.PBData(),
					SlotNumber:              slotNumber,
					BlockProposer:           blockProposer,
					ParentHeaderHash:        parentHeaderHash[:],
					PartialBlockSigningHash: partialBlockSigningHash[:],
				},
			},
		},
		MsgHash: misc.BytesToHexStr(txHash[:]),
	}
	log.Info("[BroadcastAttestationTransaction] Broadcasting Attestation Txn ",
		msg.MsgHash)
	srv.registerAndBroadcastChan <- msg
}

func (srv *Server) handleStream(s network.Stream) {

	log.Println("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	srv.addPeer <- &conn{s, true}
}

func (srv *Server) Start(keys crypto.PrivKey) (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server is already running")
	}

	srv.filter = bloom.New(200000, 5)
	//srv.chain.GetTransactionPool().SetRegisterAndBroadcastChan(srv.registerAndBroadcastChan)
	if err := srv.startListening(keys); err != nil {
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

func (srv *Server) ConnectPeer(dest string) error {
	maddr, err := multiaddr.NewMultiaddr(dest)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	// TODO: Look into PermanentAddrTTL
	srv.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	s, err := srv.host.NewStream(context.Background(),
		info.ID,
		config.GetDevConfig().ProtocolID)
	if err != nil {
		return err
	}

	srv.addPeer <- &conn{s, false}

	return nil
}

func (srv *Server) ConnectPeers() error {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	bootstrapPeers := make(map[string]bool)

	for _, multiAddr := range srv.config.User.Node.PeerList {
		//log.Info("Connecting peer ", peer)
		err := srv.peerData.AddDisconnectedPeers(multiAddr)
		if err != nil {
			log.Error("Failed to add bootstrap node in disconnected peers ", multiAddr,
				" Reason: ", err.Error())
			continue
		}
		bootstrapPeers[multiAddr] = true
	}

	peerList := make([]string, 0)
	for {
		select {
		case <-time.After(15 * time.Second):
			srv.peerInfoLock.Lock()
			if srv.inboundCount > srv.config.User.Node.MaxPeersLimit {
				srv.peerInfoLock.Unlock()
				break
			}

			maxConnectionTry := 10

			if len(peerList) == 0 {
				for _, p := range srv.peerData.DisconnectedPeers() {
					if connCount, ok := srv.ipCount[p.IP()]; ok {
						// Ignore skipping connection to addresses
						// when there is no connection with any peer
						if !(srv.totalConnections == 0 && connCount == 0) {
							continue
						}
					}
					peerList = append(peerList, p.MultiAddr())
				}
			}
			srv.peerInfoLock.Unlock()

			count := 0
			removePeers := make([]string, 0)
			for _, multiAddr := range peerList {
				if !srv.running {
					break
				}
				if count >= maxConnectionTry {
					break
				}
				log.Info("Trying to Connect ", multiAddr)
				err := srv.ConnectPeer(multiAddr)
				count += 1
				// Skip removal of bootstrapPeers
				if err != nil && !bootstrapPeers[multiAddr] {
					log.Info("Failed to connect to ", multiAddr)
					removePeers = append(removePeers, multiAddr)
					continue
				}
			}
			srv.peerInfoLock.Lock()
			for _, multiAddr := range removePeers {
				err := srv.peerData.RemovePeer(multiAddr)
				if err != nil {
					log.Error("Failed to removePeer",
						" Reason: ", err.Error())
					continue
				}
			}
			srv.peerInfoLock.Unlock()
			peerList = peerList[count:]

		case <-srv.connectPeersExit:
			return nil
		}
	}
}

func (srv *Server) startListening(keys crypto.PrivKey) error {
	multiAddrStr := fmt.Sprintf("/ip4/%s/tcp/%d",
		srv.config.User.Node.BindingIP,
		srv.config.User.Node.LocalPort)
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(multiAddrStr)

	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(keys),
	)
	if err != nil {
		return err
	}

	srv.host = host
	host.SetStreamHandler(config.GetDevConfig().ProtocolID,
		srv.handleStream)

	listenAddr := fmt.Sprintf("%s/p2p/%s",
		multiAddrStr, host.ID().Pretty())
	log.Info("Listening at ", listenAddr)

	//bindingAddress := fmt.Sprintf("%s:%d",
	//	srv.config.User.Node.BindingIP,
	//	srv.config.User.Node.LocalPort)

	//listener, err := net.Listen("tcp", bindingAddress)
	//if err != nil {
	//	return err
	//}
	//
	//srv.listener = listener
	//go srv.listenLoop(listener)

	return nil
}

func (srv *Server) run() {
	var (
		peers = make(map[string]*Peer)
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
				" addr ", c.fd.Conn().RemoteMultiaddr())
			p := newPeer(
				c.fd,
				c.inbound,
				srv.chain,
				srv.filter,
				srv.mr,
				srv.peerData,
				srv.mrDataConn,
				srv.registerAndBroadcastChan,
				srv.blockReceivedForAttestation,
				srv.attestationReceivedForBlock,
				srv.addPeerToPeerList,
				srv.blockAndPeerChan,
				srv.messagePriority)
			go srv.runPeer(p)
			peers[c.fd.ID()] = p

			ip := misc.IPFromMultiAddr(c.fd.Conn().RemoteMultiaddr().String())

			srv.ipCount[ip] += 1
			srv.totalConnections += 1
			if p.inbound {
				srv.inboundCount++
			}

			if srv.ipCount[ip] > srv.config.User.Node.MaxRedundantConnections {
				log.Info("Disconnecting due to max redundant connections")
				p.Disconnect()
				// TODO: Ban peer
			}

			srv.peerInfoLock.Unlock()
			srv.downloader.AddPeer(p)

		case pd := <-srv.delPeer:
			srv.peerInfoLock.Lock()

			log.Debug("Removing Peer", "err", pd.err)
			peer := peers[pd.stream.ID()]
			delete(peers, pd.stream.ID())
			if pd.inbound {
				srv.inboundCount--
			}
			ip := pd.ip

			srv.ipCount[ip] -= 1
			srv.totalConnections -= 1
			if pd.isPLShared {
				err := srv.peerData.AddDisconnectedPeers(pd.multiAddr)
				if err != nil {
					log.Error("Failed to add peer into disconnected peers",
						" ", pd.multiAddr,
						" Reason: ", err.Error())
				}
			}
			srv.peerInfoLock.Unlock()

			srv.downloader.RemovePeer(peer)

		case mrDataConn := <-srv.mrDataConn:
			// TODO: Process Message Receipt
			// Need to get connection too
			mrData := mrDataConn.mrData
			msgHash := misc.BytesToHexStr(mrData.Hash)
			switch mrData.Type {
			case protos.LegacyMessage_BA:
				/*
					1. Verify if Block Received for attestation is valid
					2. Broadcast the block
					3. Attest the block if Staking is Enabled on this node
				*/
				var parentHash common.Hash
				copy(parentHash[:], mrData.ParentHeaderHash)

				_, err := srv.chain.GetBlock(parentHash)
				if err != nil {
					log.Info("[BlockForAttestation] Missing Parent Block",
						" #", mrData.SlotNumber,
						" Partial Block Signing Hash ", misc.BytesToHexStr(mrData.Hash),
						" Parent Block ", misc.BytesToHexStr(mrData.ParentHeaderHash))
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
				if (finalizedHeaderHash != common.Hash{}) {
					finalizedBlock, err := srv.chain.GetBlock(finalizedHeaderHash)
					if err != nil {
						log.Error("Failed to get finalized block ",
							misc.BytesToHexStr(finalizedHeaderHash[:]))
						break
					}
					// skip slot number beyond the Finalized slot Number
					if finalizedBlock.SlotNumber() >= mrData.SlotNumber {
						log.Warn("[BlockReceived] Block #", mrData.SlotNumber,
							" is beyond finalized block #", finalizedBlock.SlotNumber())
						break
					}
				}

				var parentHash common.Hash
				copy(parentHash[:], mrData.ParentHeaderHash)

				_, err = srv.chain.GetBlock(parentHash)
				if err != nil {
					log.Info("[BlockReceived] Missing Parent Block ",
						" #", mrData.SlotNumber,
						" Block ", misc.BytesToHexStr(mrData.Hash),
						" Parent Block ", misc.BytesToHexStr(mrData.ParentHeaderHash))
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
			binMsgHash, err := misc.HexStrToBytes(registerAndBroadcast.MsgHash)
			if err != nil {
				log.Error("Error decoding message hash ", err.Error())
				continue
			}
			out := &Msg{
				msg: &protos.LegacyMessage{
					FuncName: protos.LegacyMessage_MR,
					Data: &protos.LegacyMessage_MrData{
						MrData: &protos.MRData{
							Hash: binMsgHash,
							Type: registerAndBroadcast.Msg.FuncName,
						},
					},
				},
			}
			b := registerAndBroadcast.Msg.GetBlock()
			if b != nil {
				out.msg.GetMrData().SlotNumber = b.Header.SlotNumber
				out.msg.GetMrData().ParentHeaderHash = b.Header.ParentHash
			} else {
				ba := registerAndBroadcast.Msg.GetBlockForAttestation()
				if ba != nil {
					out.msg.GetMrData().SlotNumber = ba.Block.Header.SlotNumber
					out.msg.GetMrData().ParentHeaderHash = ba.Block.Header.ParentHash
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
		msgHash := misc.BytesToHexStr(mrData.Hash)
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
			Data: &protos.LegacyMessage_MrData{
				MrData: mrData,
			},
		}
		peer.Send(out)

		time.Sleep(time.Duration(srv.config.Dev.MessageReceiptTimeout) * time.Second)
	}
}

func (srv *Server) BlockReceived(peer *Peer, b *block.Block) {
	blockHash := b.Hash()
	headerHash := misc.BytesToHexStr(blockHash[:])
	log.Info(">>> Received Block",
		" #", b.SlotNumber(),
		" Hash ", headerHash)

	// TODO: Trigger Syncing/Block downloader
	select {
	case srv.downloader.blockAndPeerChannel <- &BlockAndPeer{b, peer}:
	case <-time.After(5 * time.Second):
		log.Info("Timeout for Received Block",
			"#", b.SlotNumber(),
			"Hash", headerHash)
	}
}

func (srv *Server) UpdatePeerList(p *PeerIPWithPLData) error {
	err := srv.peerData.AddConnectedPeers(p.multiAddr)
	if err != nil {
		log.Error("Failed to Add Peer into peer list",
			" ", p.multiAddr,
			" Reason: ", err.Error())
		return err
	}
	for _, peerMultiAddr := range p.PLData.PeerIps {
		if srv.peerData.IsPeerInList(peerMultiAddr) {
			continue
		}
		err = srv.peerData.AddDisconnectedPeers(peerMultiAddr)
		if err != nil {
			log.Error("Failed to add peer ", peerMultiAddr, " in peer list",
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
		config:     config.GetConfig(),
		chain:      chain,
		ntp:        ntp.GetNTP(),
		peerData:   peerData,
		ipCount:    make(map[string]int),
		mr:         CreateMR(),
		downloader: NewDownloader(chain),

		exit:                        make(chan struct{}),
		connectPeersExit:            make(chan struct{}),
		mrDataConn:                  make(chan *MRDataConn),
		addPeerToPeerList:           make(chan *PeerIPWithPLData),
		blockAndPeerChan:            make(chan *BlockAndPeer),
		addPeer:                     make(chan *conn),
		delPeer:                     make(chan *peerDrop),
		registerAndBroadcastChan:    make(chan *messages.RegisterMessage, 100),
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
