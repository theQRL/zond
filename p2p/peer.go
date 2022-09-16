package p2p

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
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
	"github.com/theQRL/zond/transactions/pool"
	"github.com/willf/bloom"
	"io"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type MRDataConn struct {
	mrData *protos.MRData
	peer   *Peer
}

//type NodeHeaderHashWithTimestamp struct {
//	nodeHeaderHash *protos.NodeHeaderHash
//	timestamp      uint64
//}

type EBHRespInfo struct {
	Data      *protos.EpochBlockHashesResponse
	Timestamp uint64
}

type Peer struct {
	id        string
	multiAddr string
	stream    network.Stream
	inbound   bool

	lock sync.Mutex

	chain *chain.Chain

	wg                    sync.WaitGroup
	disconnectLock        sync.Mutex
	disconnected          bool
	disconnectReason      chan struct{}
	exitMonitorChainState chan struct{}
	txPool                *pool.TransactionPool
	filter                *bloom.BloomFilter // TODO: Check usage
	mr                    *MessageReceipt
	config                *config.Config
	ntp                   ntp.NTPInterface
	chainState            *protos.NodeChainState
	peerData              *metadata.PeerData

	addPeerToPeerList           chan *PeerIPWithPLData
	blockAndPeerChan            chan *BlockAndPeer
	mrDataConn                  chan *MRDataConn
	registerAndBroadcastChan    chan *messages.RegisterMessage
	blockReceivedForAttestation chan *block.Block
	attestationReceivedForBlock chan *transactions.Attest
	ebhRespInfo                 *EBHRespInfo // TODO: Add Lock before reading / writing

	inCounter           uint64
	outCounter          uint64
	lastRateLimitUpdate uint64
	bytesSent           uint64
	connectionTime      uint64
	messagePriority     map[protos.LegacyMessage_FuncName]uint64
	outgoingQueue       *PriorityQueue

	epochToBeRequested uint64 // Used by downloader to keep track of EBH request

	isPLShared bool // Flag to mark once peer list has been received by the peer
	ip         string
	publicPort string
}

func newPeer(conn network.Stream, inbound bool, chain *chain.Chain,
	filter *bloom.BloomFilter, mr *MessageReceipt,
	peerData *metadata.PeerData, mrDataConn chan *MRDataConn,
	registerAndBroadcastChan chan *messages.RegisterMessage,
	blockReceivedForAttestation chan *block.Block,
	attestationReceivedForBlock chan *transactions.Attest,
	addPeerToPeerList chan *PeerIPWithPLData,
	blockAndPeerChan chan *BlockAndPeer,
	messagePriority map[protos.LegacyMessage_FuncName]uint64) *Peer {
	p := &Peer{
		stream:                      conn,
		inbound:                     inbound,
		chain:                       chain,
		disconnected:                false,
		disconnectReason:            make(chan struct{}),
		exitMonitorChainState:       make(chan struct{}),
		txPool:                      chain.GetTransactionPool(),
		filter:                      filter,
		mr:                          mr,
		config:                      config.GetConfig(),
		ntp:                         ntp.GetNTP(),
		peerData:                    peerData,
		mrDataConn:                  mrDataConn,
		registerAndBroadcastChan:    registerAndBroadcastChan,
		blockReceivedForAttestation: blockReceivedForAttestation,
		attestationReceivedForBlock: attestationReceivedForBlock,
		addPeerToPeerList:           addPeerToPeerList,
		blockAndPeerChan:            blockAndPeerChan,
		connectionTime:              ntp.GetNTP().Time(),
		messagePriority:             messagePriority,
		outgoingQueue:               &PriorityQueue{},
	}
	p.id = p.stream.Conn().RemotePeer().Pretty()
	p.ip = misc.IPFromMultiAddr(p.stream.Conn().RemoteMultiaddr().String())

	log.Info("New Peer connected ", p.stream.Conn().RemoteMultiaddr())
	return p
}

func (p *Peer) ID() string {
	return p.id
}

func (p *Peer) IP() string {
	return p.ip
}

func (p *Peer) ChainState() *protos.NodeChainState {
	return p.chainState
}

func (p *Peer) GetTotalStakeAmount() []byte {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.chainState == nil {
		return nil
	}

	return p.chainState.TotalStakeAmount
}

func (p *Peer) GetEpochToBeRequested() uint64 {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.epochToBeRequested
}

func (p *Peer) IncreaseEpochToBeRequested() {
	p.lock.Lock()
	defer p.lock.Unlock()

	maxSlotNumber := p.chain.GetMaxPossibleSlotNumber()
	maxEpoch := maxSlotNumber / config.GetDevConfig().BlocksPerEpoch

	p.epochToBeRequested += 1

	if p.epochToBeRequested > maxEpoch {
		p.epochToBeRequested = maxEpoch
	}
}

func (p *Peer) UpdateEpochToBeRequested(epoch uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if epoch > p.epochToBeRequested {
		p.epochToBeRequested = epoch
	}
}

//func (p *Peer) GetNodeHeaderHashWithTimestamp() *NodeHeaderHashWithTimestamp {
//	return p.nodeHeaderHashWithTimestamp
//}

func (p *Peer) updateCounters() {
	timeDiff := p.ntp.Time() - p.lastRateLimitUpdate
	if timeDiff > 60 {
		p.outCounter = 0
		p.inCounter = 0
		p.lastRateLimitUpdate = p.ntp.Time()
	}
}

func (p *Peer) SendEBHReq(epoch uint64, finalizedHeaderHash []byte) error {
	p.UpdateEpochToBeRequested(epoch)

	msg := &Msg{
		msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_EBHREQ,
			Data: &protos.LegacyMessage_EpochBlockHashesRequest{
				EpochBlockHashesRequest: &protos.EpochBlockHashesRequest{
					Epoch:               p.GetEpochToBeRequested(),
					FinalizedHeaderHash: finalizedHeaderHash,
				},
			},
		},
	}
	return p.Send(msg)
}

func (p *Peer) Send(msg *Msg) error {
	priority, ok := p.messagePriority[msg.msg.FuncName]
	if !ok {
		log.Warn("Unexpected FuncName while SEND",
			"FuncName", msg.msg.FuncName)
		return nil
	}
	outgoingMsg := CreateOutgoingMessage(priority, msg.msg)
	if p.outgoingQueue.Full() {
		log.Info("Outgoing Queue Full: Skipping Message")
		return errors.New("disconnecting: Outgoing Queue Full")
	}
	p.outgoingQueue.Push(outgoingMsg)
	return p.SendNext()
}

func (p *Peer) SendNext() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.disconnected {
		return errors.New("peer disconnected")
	}

	p.updateCounters()
	if float32(p.outCounter) >= float32(p.config.User.Node.PeerRateLimit)*0.9 {
		log.Info("Send Next Cancelled as",
			"p.outcounter", p.outCounter,
			"rate limit", float32(p.config.User.Node.PeerRateLimit)*0.9)
		return nil
	}

	for p.bytesSent < p.config.Dev.MaxBytesOut {
		data := p.outgoingQueue.Pop()
		if data == nil {
			return nil
		}
		om := data.(*OutgoingMessage)
		outgoingBytes, _ := om.bytesMessage, om.msg

		if outgoingBytes == nil {
			log.Info("Outgoing bytes Nil")
			return nil
		}
		p.bytesSent += uint64(len(outgoingBytes))
		_, err := p.stream.Write(outgoingBytes)

		if err != nil {
			log.Error("Error while writing message on socket", "error", err)
			p.Disconnect()
			return nil
		}
	}
	if p.bytesSent >= p.config.Dev.MaxBytesOut {
		return errors.New("BytesSent >= MaxBytesOut")
	}

	return nil
}

func (p *Peer) ReadMsg() (msg *Msg, size uint32, err error) {
	// TODO: Add Read timeout
	msg = &Msg{}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(p.stream, buf); err != nil {
		return msg, 0, err
	}
	size = misc.ConvertBytesToLong(buf)
	buf = make([]byte, size)
	if _, err := io.ReadFull(p.stream, buf); err != nil {
		return nil, 0, err
	}
	message := &protos.LegacyMessage{}
	err = proto.Unmarshal(buf, message)
	msg.msg = message
	return msg, size + 4, err // 4 Byte Added for MetaData that includes the size of actual data
}

func (p *Peer) readLoop() {
	p.wg.Add(1)
	defer p.wg.Done()

	for {
		p.updateCounters()
		totalBytesRead := uint32(0)
		msg, size, err := p.ReadMsg()
		if err != nil {
			p.Disconnect()
			return
		}
		msg.ReceivedAt = time.Now()
		if err = p.handle(msg); err != nil {
			log.Info("Error at handle message")
			p.Disconnect()
			return
		}
		p.inCounter += 1
		if float32(p.inCounter) > 2.2*float32(p.config.User.Node.PeerRateLimit) {
			log.Warn("Rate Limit Hit")
			p.Disconnect()
			return
		}

		totalBytesRead += size
		if msg.msg.FuncName != protos.LegacyMessage_P2P_ACK {
			p2pAck := &protos.P2PAcknowledgement{
				BytesProcessed: totalBytesRead,
			}
			out := &Msg{}
			out.msg = &protos.LegacyMessage{
				FuncName: protos.LegacyMessage_P2P_ACK,
				Data: &protos.LegacyMessage_P2PAckData{
					P2PAckData: p2pAck,
				},
			}
			err = p.Send(out)
			if err != nil {
				p.Disconnect()
			}
		}
	}
}

func (p *Peer) monitorChainState() {
	p.wg.Add(1)
	defer p.wg.Done()
	for {
		log.Debug("Monitor Chain State running for ", p.IP(), " ", p.ID())
		select {
		case <-time.After(30 * time.Second):
			currentTime := p.ntp.Time()
			delta := int64(currentTime)
			if p.chainState != nil {
				delta -= int64(p.chainState.Timestamp)
			} else {
				delta -= int64(p.connectionTime)
			}
			if delta > int64(p.config.User.ChainStateTimeout) {
				log.Warn("Disconnecting Peer due to Ping Timeout",
					" delta ", delta,
					" currentTime ", currentTime,
					" peer ", p.IP(), " ", p.ID())
				p.Disconnect()
				return
			}

			lastBlock := p.chain.GetLastBlock()
			lastBlockMetaData, err := p.chain.GetBlockMetaData(lastBlock.Hash())
			if err != nil {
				log.Warn("Ping Failed Disconnecting ", p.stream.Conn().RemoteMultiaddr())
				p.Disconnect()
				return
			}
			lastBlockHash := lastBlock.Hash()
			chainStateData := &protos.NodeChainState{
				SlotNumber:       lastBlock.SlotNumber(),
				HeaderHash:       lastBlockHash[:],
				TotalStakeAmount: lastBlockMetaData.TotalStakeAmount(),
				Version:          p.config.Dev.Version,
				Timestamp:        p.ntp.Time(),
			}
			out := &Msg{}
			out.msg = &protos.LegacyMessage{
				FuncName: protos.LegacyMessage_CHAINSTATE,
				Data: &protos.LegacyMessage_ChainStateData{
					ChainStateData: chainStateData,
				},
			}

			err = p.Send(out)
			if err != nil {
				log.Info("Error while sending ChainState",
					p.stream.Conn().RemoteMultiaddr())
				p.Disconnect()
				return
			}

			if p.chainState == nil {
				log.Debug("Ignoring MonitorState check as peer chain state is nil for ", p.IP(), " ", p.ID())
				continue
			}
		case <-p.exitMonitorChainState:
			return
		}
	}
}

func (p *Peer) handle(msg *Msg) error {
	/*
		Error returned by handle, result into disconnection.
		In some cases, like when peer receives txn hash which already
		present with node will result into failure while adding txn
		and thus may result into disconnection.

		Error should not be returned until the above cases, has been handled.
	*/
	switch msg.msg.FuncName {

	case protos.LegacyMessage_VE:
		log.Debug("Received VE MSG")
		if msg.msg.GetVeData() == nil {
			out := &Msg{}
			veData := &protos.VEData{
				Version:         "",
				GenesisPrevHash: []byte("0"),
				RateLimit:       100,
			}
			out.msg = &protos.LegacyMessage{
				FuncName: protos.LegacyMessage_VE,
				Data: &protos.LegacyMessage_VeData{
					VeData: veData,
				},
			}
			err := p.Send(out)
			return err
		}
		veData := msg.msg.GetVeData()
		log.Info("", "version:", veData.Version,
			"GenesisPrevHash:", veData.GenesisPrevHash, "RateLimit:", veData.RateLimit)

	case protos.LegacyMessage_PL:
		log.Debug("Received PL MSG")
		if p.isPLShared {
			log.Debug("Peer list already shared before")
			return nil
		}
		p.publicPort = strconv.FormatUint(uint64(msg.msg.GetPlData().PublicPort), 10)

		p.multiAddr = fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", p.ip, p.publicPort, p.stream.Conn().RemotePeer())
		p.isPLShared = true
		p.addPeerToPeerList <- &PeerIPWithPLData{p.multiAddr, msg.msg.GetPlData()}

	case protos.LegacyMessage_PONG:
		log.Debug("Received PONG MSG")

	case protos.LegacyMessage_MR:
		mrData := msg.msg.GetMrData()
		mrDataConn := &MRDataConn{
			mrData,
			p,
		}
		p.mrDataConn <- mrDataConn

	case protos.LegacyMessage_SFM:
		mrData := msg.msg.GetMrData()
		msg := p.mr.Get(mrData.Hash)
		if msg != nil {
			out := &Msg{}
			out.msg = msg
			p.Send(out)
		}

	case protos.LegacyMessage_BA:
		ba := msg.msg.GetBlockForAttestation()
		p.HandleBlockForAttestation(ba.Block, ba.Signature)

	case protos.LegacyMessage_BK:
		b := msg.msg.GetBlock()
		p.HandleBlock(b)

	case protos.LegacyMessage_EBHREQ:
		epochHeaderHashResp := &protos.EpochBlockHashesResponse{
			IsHeaderHashFinalized: true,
		}

		ebhReq := msg.msg.GetEpochBlockHashesRequest()
		var finalizedHeaderHash common.Hash
		copy(finalizedHeaderHash[:], ebhReq.FinalizedHeaderHash)
		b, err := p.chain.GetBlock(finalizedHeaderHash)
		if err != nil {
			epochHeaderHashResp.IsHeaderHashFinalized = false
		}

		if b != nil && b.SlotNumber() > 0 {
			parentBlockMetaData, err := p.chain.GetBlockMetaData(b.ParentHash())
			if err != nil {
				log.Error("Block found but parent Block MetaData not found ", err.Error())
				return nil
			}
			if !reflect.DeepEqual(
				parentBlockMetaData.FinalizedChildHeaderHash(), ebhReq.FinalizedHeaderHash) {
				epochHeaderHashResp.IsHeaderHashFinalized = false
			}
		}

		if epochHeaderHashResp.IsHeaderHashFinalized {
			epoch := ebhReq.Epoch
			epochBlockHashes, err := p.chain.GetEpochHeaderHashes(epoch)
			if err != nil {
				log.Error("Error in GetEpochHeaderHashes")
				return nil
			}

			epochHeaderHashResp.EpochBlockHashesMetaData = epochBlockHashes
		}
		out := &Msg{}
		out.msg = &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_EBHRESP,
			Data: &protos.LegacyMessage_EpochBlockHashesResponse{
				EpochBlockHashesResponse: epochHeaderHashResp,
			},
		}
		p.Send(out)
	case protos.LegacyMessage_EBHRESP:
		data := msg.msg.GetEpochBlockHashesResponse()
		p.ebhRespInfo = &EBHRespInfo{
			Data:      data,
			Timestamp: p.ntp.Time(),
		}
		// store the requested headerhash
		// store the requested timestamp
		// store response locally
		// Compare requested timestamp with current timestamp
		// as it will be accessed by downloader itself
		// after certain threshold
		// Use lock while writing the data to the variable
	case protos.LegacyMessage_FB:
		fbData := msg.msg.GetFbData()
		var blockHeaderHash common.Hash
		copy(blockHeaderHash[:], fbData.BlockHeaderHash)
		log.Info("Fetch Block Request",
			" BlockHeaderHash ", misc.BytesToHexStr(blockHeaderHash[:]),
			" Peer ", p.stream.Conn().RemoteMultiaddr())

		b, err := p.chain.GetBlock(blockHeaderHash)
		if err != nil {
			log.Info("Disconnecting Peer, as GetBlock returned nil")
			return errors.New("peer protocol error")
		}
		pbData := &protos.PBData{
			Block: b.PBData(),
		}
		out := &Msg{}
		out.msg = &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_PB,
			Data: &protos.LegacyMessage_PbData{
				PbData: pbData,
			},
		}
		p.Send(out)

	case protos.LegacyMessage_PB:
		pbData := msg.msg.GetPbData()
		if pbData.Block == nil {
			log.Info("Disconnecting Peer, as no block sent for Push Block")
			return errors.New("peer protocol error")
		}

		b := block.BlockFromPBData(pbData.Block)
		p.blockAndPeerChan <- &BlockAndPeer{b, p}

	case protos.LegacyMessage_TT: // Transfer Token Transaction
		p.HandleTransaction(msg, msg.msg.GetTtData())
	case protos.LegacyMessage_ST: // Slave Transaction
		p.HandleTransaction(msg, msg.msg.GetStData())
	case protos.LegacyMessage_AT: // Attest Transaction
		p.HandleAttestTransaction(msg, msg.msg.GetAtData())
	case protos.LegacyMessage_SYNC:
		//log.Warn("SYNC has not been Implemented <<<< --- ")
	case protos.LegacyMessage_CHAINSTATE:
		chainStateData := msg.msg.GetChainStateData()
		p.HandleChainState(chainStateData)

	case protos.LegacyMessage_P2P_ACK:
		p2pAckData := msg.msg.GetP2PAckData()
		p.bytesSent -= uint64(p2pAckData.BytesProcessed)
		if p.bytesSent < 0 {
			log.Warn("Disconnecting Peer due to negative bytes sent",
				" bytesSent ", p.bytesSent,
				" BytesProcessed ", p2pAckData.BytesProcessed)
			return errors.New("peer protocol error")
		}
		return p.SendNext()
	}
	return nil
}

func (p *Peer) HandleBlockForAttestation(pbBlock *protos.Block, signature []byte) {
	b := block.BlockFromPBData(pbBlock)
	partialBlockSigningHash := b.PartialBlockSigningHash()
	if !p.mr.IsRequested(partialBlockSigningHash, p) {
		log.Error("Unrequested Block Received for Attestation from ", p.IP(), " ", p.ID(),
			" #", b.SlotNumber(),
			" PartialBlockSigningHash ", misc.BytesToHexStr(partialBlockSigningHash[:]))
		return
	}
	log.Info("Received Block for Attestation from ", p.IP(), " ", p.ID(),
		" #", b.SlotNumber(),
		" PartialBlockSigningHash ", misc.BytesToHexStr(partialBlockSigningHash[:]))

	// TODO: Add Block Validation

	msg := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			FuncName: protos.LegacyMessage_BA,
			Data: &protos.LegacyMessage_BlockForAttestation{
				BlockForAttestation: &protos.BlockForAttestation{
					Block:     b.PBData(),
					Signature: signature,
				},
			},
		},
		MsgHash: misc.BytesToHexStr(partialBlockSigningHash[:]),
	}
	p.registerAndBroadcastChan <- msg

	p.blockReceivedForAttestation <- b

}

func (p *Peer) HandleBlock(pbBlock *protos.Block) {
	// TODO: Validate Message
	b := block.BlockFromPBData(pbBlock)
	hash := b.Hash()
	expectedHash := block.ComputeBlockHash(b)
	if hash != expectedHash {
		log.Error("Invalid block hash", "Expected hash", expectedHash, "Found hash", hash, "Block #", b.SlotNumber())
		return
	}
	if !p.mr.IsRequested(b.Hash(), p) {
		log.Error("Unrequested Block Received from ", p.IP(), " ", p.ID(), " #", b.SlotNumber(), " ",
			misc.BytesToHexStr(hash[:]))
		return
	}
	log.Info("Received Block from ", p.IP(), " ", p.ID(), " #", b.SlotNumber(), " ",
		misc.BytesToHexStr(hash[:]))

	if !p.chain.AddBlock(b) {
		log.Warn("Failed To Add Block")
		return
	}

	msg := &protos.LegacyMessage{
		Data: &protos.LegacyMessage_Block{
			Block: b.PBData(),
		},
		FuncName: protos.LegacyMessage_BK,
	}

	registerMessage := &messages.RegisterMessage{
		MsgHash: misc.BytesToHexStr(hash[:]),
		Msg:     msg,
	}

	select {
	case p.registerAndBroadcastChan <- registerMessage:
	case <-time.After(10 * time.Second):
		log.Warn("[HandleBlock] RegisterAndBroadcastChan Timeout ",
			p.IP(), " ", p.ID())
	}
}

func (p *Peer) HandleTransaction(msg *Msg, txData *protos.Transaction) error {
	tx := transactions.ProtoToTransaction(txData)
	txHash := tx.Hash()

	if !p.mr.IsRequested(txHash, p) {
		log.Warn("[HandleTransaction] Received Unrequested txn ",
			" Peer", p.IP(), " ", p.ID(),
			" Tx Hash", misc.BytesToHexStr(txHash[:]))
		return nil
	}

	if err := p.chain.ValidateTransaction(txData); err != nil {
		return nil
	}
	err := p.txPool.Add(tx, txHash, p.chain.GetLastBlock().SlotNumber(), p.ntp.Time())
	if err != nil {
		log.Error("Error while adding TransferTxn into TxPool",
			"Txhash", txHash,
			"Error", err.Error())
		return err
	}

	msg2 := &protos.LegacyMessage{
		FuncName: msg.msg.FuncName,
		Data:     msg.msg.Data,
	}
	registerMessage := &messages.RegisterMessage{
		MsgHash: misc.BytesToHexStr(txHash[:]),
		Msg:     msg2,
	}
	select {
	case p.registerAndBroadcastChan <- registerMessage:
	case <-time.After(10 * time.Second):
		log.Warn("[TX] RegisterAndBroadcastChan Timeout ",
			p.IP(), " ", p.ID())
	}
	return nil
}

func (p *Peer) HandleAttestTransaction(msg *Msg, txData *protos.ProtocolTransactionData) error {
	pbData := txData.Tx
	tx := transactions.ProtoToProtocolTransaction(pbData)
	var partialBlockSigningHash common.Hash
	copy(partialBlockSigningHash[:], txData.PartialBlockSigningHash)
	txHash := tx.TxHash(tx.GetSigningHash(partialBlockSigningHash))

	if !p.mr.IsRequested(txHash, p) {
		log.Warn("[HandleAttestTransaction] Received Unrequested txn",
			" Peer", p.IP(), " ", p.ID(),
			" Tx Hash", misc.BytesToHexStr(txHash[:]))
		return nil
	}

	var parentBlockHash common.Hash
	copy(parentBlockHash[:], txData.ParentHeaderHash)

	parentMetaData, err := p.chain.GetBlockMetaData(parentBlockHash)
	if err != nil {
		log.Warn("failed to get parent block metadata",
			" Peer", p.IP(), " ", p.ID(),
			" Tx Hash", misc.BytesToHexStr(txHash[:]))
		return nil
	}

	slotValidatorsMetaData, err := p.chain.GetSlotValidatorsMetaDataBySlotNumber(parentMetaData.TrieRoot(), txData.SlotNumber, parentBlockHash)
	if err != nil {
		log.Error("Error getting validators type")
		return nil
	}

	if err := p.chain.ValidateAttestTransaction(pbData, slotValidatorsMetaData, partialBlockSigningHash, txData.SlotNumber, parentMetaData.SlotNumber()); err != nil {
		log.Error("[HandleAttestTransaction] Attest Transaction Validation Failed ", err)
		return nil
	}
	p.attestationReceivedForBlock <- tx.(*transactions.Attest)

	msg2 := &protos.LegacyMessage{
		FuncName: msg.msg.FuncName,
		Data:     msg.msg.Data,
	}
	registerMessage := &messages.RegisterMessage{
		MsgHash: misc.BytesToHexStr(txHash[:]),
		Msg:     msg2,
	}
	select {
	case p.registerAndBroadcastChan <- registerMessage:
	case <-time.After(10 * time.Second):
		log.Warn("[AT] RegisterAndBroadcastChan Timeout ",
			p.IP(), " ", p.ID())
	}
	return nil
}

func (p *Peer) HandleChainState(nodeChainState *protos.NodeChainState) {
	p.chainState = nodeChainState
	p.chainState.Timestamp = p.ntp.Time()
}

func (p *Peer) SendFetchBlock(blockHeaderHash common.Hash) error {
	log.Info("Fetching",
		" Block ", misc.BytesToHexStr(blockHeaderHash[:]),
		" Peer ", p.stream.Conn().RemoteMultiaddr())
	out := &Msg{}
	fbData := &protos.FBData{
		BlockHeaderHash: blockHeaderHash[:],
	}
	out.msg = &protos.LegacyMessage{
		FuncName: protos.LegacyMessage_FB,
		Data: &protos.LegacyMessage_FbData{
			FbData: fbData,
		},
	}
	return p.Send(out)
}

func (p *Peer) SendPeerList() {
	peerList := p.peerData.PeerList()
	out := &Msg{}
	plData := &protos.PLData{
		PeerIps:    peerList,
		PublicPort: uint32(config.GetUserConfig().Node.PublicPort),
	}
	out.msg = &protos.LegacyMessage{
		FuncName: protos.LegacyMessage_PL,
		Data: &protos.LegacyMessage_PlData{
			PlData: plData,
		},
	}
	p.Send(out)
}

func (p *Peer) SendVersion() {
	out := &Msg{}
	veData := &protos.VEData{
		Version:         p.config.Dev.Version,
		GenesisPrevHash: p.config.Dev.Genesis.GenesisPrevHeaderHash[:],
		RateLimit:       p.config.User.Node.PeerRateLimit,
	}
	out.msg = &protos.LegacyMessage{
		FuncName: protos.LegacyMessage_PL,
		Data: &protos.LegacyMessage_VeData{
			VeData: veData,
		},
	}
	p.Send(out)
}

func (p *Peer) SendSync() {
	out := &Msg{}
	syncData := &protos.SYNCData{
		State: "Synced",
	}
	out.msg = &protos.LegacyMessage{
		FuncName: protos.LegacyMessage_SYNC,
		Data: &protos.LegacyMessage_SyncData{
			SyncData: syncData,
		},
	}
	p.Send(out)
}

func (p *Peer) handshake() {
	p.SendPeerList()
	// p.SendVersion()
	p.SendSync()
}

func (p *Peer) run() (remoteRequested bool) {
	p.handshake()
	go p.readLoop()
	go p.monitorChainState()

loop:
	for {
		select {
		case <-p.disconnectReason:
			break loop
		}
	}
	p.close()
	p.wg.Wait()

	log.Info("Peer routine closed for ", p.stream.Conn().RemoteMultiaddr())
	return remoteRequested
}

func (p *Peer) close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	log.Info("Disconnected ", p.stream.Conn().RemoteMultiaddr())

	close(p.exitMonitorChainState)
	p.stream.Close()
}

func (p *Peer) Disconnect() {
	p.disconnectLock.Lock()
	defer p.disconnectLock.Unlock()

	if !p.disconnected {
		p.disconnected = true
		log.Info("Disconnecting ", p.stream.Conn().RemoteMultiaddr())
		p.disconnectReason <- struct{}{}
	}
}
