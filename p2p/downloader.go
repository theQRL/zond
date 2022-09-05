package p2p

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/protos"
	"math/big"
	"sort"
	"sync"
	"time"
)

const MaxRequest = 40 // Size has to be calculated based on maximum possible values on Queue

type BlockAndPeer struct {
	block *block.Block
	peer  *Peer
}

type HashesAndPeerInfo struct {
	slotNumber      uint64
	peerByBlockHash map[string][]*Peer
}

type RequestTracker struct {
	lock sync.Mutex

	requests map[string]*Peer
	sequence []string // Sequence of Header hashes in which request was made
}

func (r *RequestTracker) AddPeerRequest(headerHash string, peer *Peer) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.requests[headerHash] = peer
	r.sequence = append(r.sequence, headerHash)
}

func (r *RequestTracker) GetPeerByHeaderHash(headerHash string) (*Peer, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	data, ok := r.requests[headerHash]
	return data, ok
}

func (r *RequestTracker) RemoveRequestKey(headerHash string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.requests, headerHash)
}

func (r *RequestTracker) TotalRequest() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.sequence)
}

func (r *RequestTracker) GetSequenceByIndex(index int) string {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.sequence[index]
}

func (r *RequestTracker) RemoveFirstElementFromSequence() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.sequence = r.sequence[1:]
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		requests: make(map[string]*Peer),
		sequence: make([]string, 0),
	}
}

type Downloader struct {
	lock sync.Mutex

	isSyncing bool
	chain     *chain.Chain
	ntp       ntp.NTPInterface

	blockAndPeerChannel chan *BlockAndPeer

	wg                     sync.WaitGroup
	exitDownloadMonitor    chan struct{}
	peersList              map[string]*Peer
	ignorePeers            map[string]int  // List of peers that need to be ignored
	requestTracker         *RequestTracker // Maintains block header hash requested by the peer
	peerSelectionCount     int
	slotNumberProcessed    chan uint64
	done                   chan struct{}
	consumerRunning        bool
	blockDownloaderRunning bool
}

/*

When Downloader is triggered.
It is triggered when
1> a peer shows a chain state with
same slot number but different hash

2> a peer shows a chain state with
atleast +3 slot number

3> a peer shows a chain state with
lower slot number

===================================
1> the ChainState of a peer shows
current header hash is not a part of the
Current node chain

*/

/*
TODO:
1. Request list of child header hashes from other nodes,
   from the header hash provided by current node

   Response should contain SlotNumber and headerhash
2. Prepare mapping based on Response received by multiple peers
   Map key SlotNumber, value headerhashes
   Map key headerhash, value peers

   Prepare list of slotnumbers in the sequential order of downloading
3. Each time a block is received by Downloader,
   remove slot number, header hash keys from map.
*/

func (d *Downloader) AddPeer(p *Peer) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if _, ok := d.peersList[p.ID()]; ok {
		return
	}

	d.peersList[p.ID()] = p
}

func (d *Downloader) RemovePeer(p *Peer) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	if _, ok := d.peersList[p.ID()]; !ok {
		return false
	}

	delete(d.peersList, p.ID())
	return true
}

//func (d *Downloader) GetTargetPeerCount() int {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	return len(d.targetPeers)
//}
//
//func (d *Downloader) GetTargetPeerByID(id string) (*TargetNode, bool) {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	data, ok := d.targetPeers[id]
//	return data, ok
//}
//
//func (d *Downloader) GetTargetPeerLength() int {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	return len(d.targetPeers)
//}
//
//func (d *Downloader) GetRandomTargetPeer() *TargetNode {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	length := len(d.targetPeers)
//	if length == 0 {
//		return nil
//	}
//
//	randIndex := rand.Intn(length)
//
//	for _, targetPeer := range d.targetPeers {
//		if randIndex == 0 {
//			return targetPeer
//		}
//		randIndex--
//	}
//
//	return nil
//}

func (d *Downloader) isDownloaderRunning() bool {
	return d.consumerRunning || d.blockDownloaderRunning
}

//func (d *Downloader) AddPeerToTargetPeers(p *Peer) {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	if _, ok := d.targetPeers[p.ID()]; ok {
//		return
//	}
//
//	d.targetPeers[p.ID()] = &TargetNode{
//		peer: p,
//		requestedBlockNumbers: make(map[uint64]bool),
//	}
//	d.targetPeerList = append(d.targetPeerList, p.ID())
//}
//
//func (d *Downloader) RemovePeerFromTargetPeers(p *Peer) bool {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	if _, ok := d.targetPeers[p.ID()]; !ok {
//		return false
//	}
//
//	delete(d.targetPeers, p.ID())
//	for i, targetPeerID := range d.targetPeerList {
//		if targetPeerID == p.ID() {
//			d.targetPeerList = append(d.targetPeerList[:i], d.targetPeerList[i+1:]...)
//			return true
//		}
//	}
//	return false
//}

func (d *Downloader) Consumer(lastBlockHeaderToDownload common.Hash, peerGroup []*Peer) {
	d.consumerRunning = true
	defer func() {
		d.consumerRunning = false
	}()
	pendingBlocks := make(map[string]*block.Block)
	for {
		select {
		case blockAndPeer := <-d.blockAndPeerChannel:
			b := blockAndPeer.block
			// Ensure if the block received is from the same Peer from which it was requested
			bHash := blockAndPeer.block.Hash()
			strHeaderHash := misc.BytesToHexStr(bHash[:])
			targetPeer, ok := d.requestTracker.GetPeerByHeaderHash(strHeaderHash)
			if !ok || blockAndPeer.peer.ID() != targetPeer.ID() {
				continue
			}

			pendingBlocks[strHeaderHash] = b

			for d.isSyncing && d.requestTracker.TotalRequest() > 0 {
				strHeaderHash := d.requestTracker.GetSequenceByIndex(0)
				b, ok := pendingBlocks[strHeaderHash]
				if !ok {
					break
				}
				log.Info("Trying To Add Block",
					" #", b.SlotNumber(),
					" ", misc.BytesToHexStr(bHash[:]))
				delete(pendingBlocks, strHeaderHash)
				d.requestTracker.RemoveFirstElementFromSequence()
				d.requestTracker.RemoveRequestKey(strHeaderHash)

				if !d.chain.AddBlock(b) {
					log.Warn("Failed To Add Block")
					break
				}
				d.slotNumberProcessed <- b.SlotNumber()
			}

			if d.isSyncingFinished(false, lastBlockHeaderToDownload, peerGroup) {
				log.Info("Block Download Syncing Finished")
				return
			}
		case <-time.After(30 * time.Second):
			log.Info("[Consumer Timeout] Ignoring Peer")
			d.isSyncingFinished(true, lastBlockHeaderToDownload, nil)
			return
		case <-d.done:
			d.isSyncing = false
			return
		}
	}
}

func (d *Downloader) RequestForBlock(targetSlotNumbers []*HashesAndPeerInfo,
	nextIndexForRequest int, numberOfRequests int) (int, int, error) {
	log.Info("Requesting For Blocks")

main:
	for ; nextIndexForRequest < len(targetSlotNumbers); nextIndexForRequest++ {
		hashesPeerInfo := targetSlotNumbers[nextIndexForRequest]
		for headerHash := range hashesPeerInfo.peerByBlockHash {
			binData, err := misc.HexStrToBytes(headerHash)
			var binHeaderHash common.Hash
			copy(binHeaderHash[:], binData)
			if err != nil {
				continue
			}
			b, err := d.chain.GetBlock(binHeaderHash)

			// Ignore making request if block already exists
			if err == nil && b != nil {
				continue
			}

			peerLen := len(hashesPeerInfo.peerByBlockHash[headerHash])

			for peerLen > 0 {
				selectedPeer := (hashesPeerInfo.slotNumber) % uint64(peerLen)
				peer := hashesPeerInfo.peerByBlockHash[headerHash][selectedPeer]
				hashesPeerInfo.peerByBlockHash[headerHash] = hashesPeerInfo.peerByBlockHash[headerHash][1:]
				peerLen--
				if peer == nil || peer.disconnected {
					continue
				}
				err := peer.SendFetchBlock(binHeaderHash)
				if err != nil {
					log.Error("Error fetching block from peer ", peer.ID())
					continue
				}
				d.requestTracker.AddPeerRequest(headerHash, peer)
				numberOfRequests++
				break main
			}
		}
	}

	return numberOfRequests, nextIndexForRequest, nil
}

func (d *Downloader) BlockDownloader(targetSlotNumbers []*HashesAndPeerInfo,
	lastBlockHeaderToDownload common.Hash, peerGroup []*Peer) {
	d.blockDownloaderRunning = true
	defer func() {
		d.blockDownloaderRunning = false
	}()

	log.Info("Block Downloader Started")

	nextIndexForRequest := 0
	numberOfRequests, nextIndexForRequest, err := d.RequestForBlock(targetSlotNumbers, nextIndexForRequest,
		0)
	if err != nil {
		log.Error("Error while Requesting for block",
			"Block #", targetSlotNumbers[nextIndexForRequest].slotNumber,
			"Error", err.Error())
	}
	if numberOfRequests == 0 {
		d.isSyncingFinished(false, lastBlockHeaderToDownload, peerGroup)
		return
	}
	targetSlotNumbersLen := len(targetSlotNumbers)
	for {
		select {
		case blockNumber := <-d.slotNumberProcessed:
			numberOfRequests -= 1
			for numberOfRequests < MaxRequest-10 && nextIndexForRequest < targetSlotNumbersLen {
				numberOfRequests, nextIndexForRequest, err = d.RequestForBlock(targetSlotNumbers, nextIndexForRequest,
					numberOfRequests)
				if err != nil {
					log.Error("Error while Requesting for block",
						"Block #", blockNumber,
						"Error", err.Error())
					continue
				}
			}
		case <-d.done:
			d.isSyncing = false
			log.Info("Producer Exits")
			return
		}
	}
}

//func (d *Downloader) NewTargetNode(nodeHeaderHash *protos.NodeHeaderHash, peer *Peer) {
//	d.targetNode = &TargetNode {
//		peer: peer,
//		blockNumber: nodeHeaderHash.SlotNumber,
//		headerHashes: nodeHeaderHash.Headerhashes,
//		nodeHeaderHash: nodeHeaderHash,
//		lastRequestedBlockNumber: nodeHeaderHash.SlotNumber,
//
//		requestedBlockNumbers: make(map[uint64]bool),
//	}
//}

func (d *Downloader) isSyncingFinished(forceFinish bool,
	lastBlockHeaderToDownload common.Hash, peerGroup []*Peer) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.isSyncing {
		return true
	}

	if forceFinish {
		d.isSyncing = false
		log.Info("Syncing FINISHED")
		close(d.done)
		return true
	}
	b, err := d.chain.GetBlock(lastBlockHeaderToDownload)

	if err == nil && b != nil {
		for _, peer := range peerGroup {
			peer.IncreaseEpochToBeRequested()
		}
		d.isSyncing = false
		log.Info("Syncing FINISHED")
		close(d.done)
		return true
	}
	return false
}

func NewDownloader(c *chain.Chain) (d *Downloader) {
	d = &Downloader{
		isSyncing: false,
		chain:     c,
		ntp:       ntp.GetNTP(),

		peersList:           make(map[string]*Peer),
		peerSelectionCount:  0,
		slotNumberProcessed: make(chan uint64, MaxRequest),
		blockAndPeerChannel: make(chan *BlockAndPeer, MaxRequest*2),
		done:                make(chan struct{}),
	}
	return
}

func (d *Downloader) Exit() {
	log.Debug("Shutting Down Downloader")
	// TODO: Check if done is already closed
	if d.isSyncing {
		close(d.done)
	}
	close(d.exitDownloadMonitor)
	d.wg.Wait()
}

func (d *Downloader) AddEpochBlockHashesResponse(data *protos.EpochBlockHashesResponse, peer *Peer) {

}

func (d *Downloader) DownloadMonitor() {
	log.Info("Running download monitor")
	d.exitDownloadMonitor = make(chan struct{})
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		select {
		case <-time.After(30 * time.Second):
			// Ignore if Consumer or BlockDownloader already running.
			if d.isDownloaderRunning() {
				continue
			}
			if d.isSyncing {
				continue
			}
			startingNonFinalizedEpoch, err := d.chain.GetStartingNonFinalizedEpoch()
			if err != nil {
				log.Error("Unable to Initialize Downloader")
				log.Error("Failed to get Finalized Epoch ", err.Error())
				continue
			}

			addedPeers := make(map[string]uint64) // Temporary variable to ensure
			// duplicate peers are not added
			peerGroup := make([]*Peer, 0)
			totalStakeAmount, err := d.chain.GetTotalStakeAmount()
			if err != nil {
				log.Error("[DownloadMonitor] Failed to GetTotalStakeAmount ",
					err.Error())
				continue
			}
			for _, p := range d.peersList {
				// check if peer already exists
				if _, ok := addedPeers[p.ID()]; ok {
					continue
				}
				peerChainState := p.ChainState()
				if peerChainState == nil {
					continue
				}
				peerTotalStakeAmount := big.NewInt(0)
				err := peerTotalStakeAmount.UnmarshalText(peerChainState.TotalStakeAmount)
				if err != nil {
					log.Error("[DownloadMonitor] Failed to unmarshal TotalStakeAmount for ",
						" Peer ", p.ID())
					continue
				}
				// Ignore peer as current total stake amount of the chain
				// is higher than the peer's chain
				if totalStakeAmount.Cmp(peerTotalStakeAmount) == 1 {
					continue
				}

				epoch := peerChainState.SlotNumber / config.GetDevConfig().BlocksPerEpoch
				if epoch < startingNonFinalizedEpoch {
					continue
				}

				// If block not found, then syncing is required
				var headerHash common.Hash
				copy(headerHash[:], peerChainState.HeaderHash)
				if _, err := d.chain.GetBlock(headerHash); err != nil {
					peerGroup = append(peerGroup, p)
					addedPeers[p.ID()] = 0
				}
			}
			if len(peerGroup) == 0 {
				continue
			}
			finalizedHeaderHash, err := d.chain.GetFinalizedHeaderHash()
			if err != nil {
				log.Error("[Downloader] Failed to Get Finalized Header Hash ", err.Error())
				continue
			}
			/*
				Select group with highest number of peers and
				consider them as target peer

			*/
			log.Info("===================== downloading ====================")
			d.Initialize(peerGroup, startingNonFinalizedEpoch, finalizedHeaderHash)
		case <-d.exitDownloadMonitor:
			return
		}
	}
}

func (d *Downloader) Initialize(peerGroup []*Peer,
	startingNonFinalizedEpoch uint64, finalizedHeaderHash common.Hash) {
	/*
		Syncing start after finalized block
		1. Request for all child header hashes from all peers of an epoch.
		2. Assign header hashes with Peer
		3. Request for block from randomly selected peer
	*/
	log.Info("Initializing Downloader")
	requestTimestamp := d.ntp.Time()
	for _, peer := range peerGroup {
		peer.SendEBHReq(startingNonFinalizedEpoch, finalizedHeaderHash[:])
	}
	d.isSyncing = true
	d.done = make(chan struct{})

	// Delay 10 seconds to receive hashes from peers
	select {
	case <-time.After(10 * time.Second):
	}

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	minimumStartSlotNumber := startingNonFinalizedEpoch * blocksPerEpoch
	hashesWithPeerInfoBySlotNumber := make(map[uint64]*HashesAndPeerInfo)

	for i := 0; i < len(peerGroup); i++ {
		peer := peerGroup[i]
		ebhRespInfo := peer.ebhRespInfo
		if peer.disconnected || ebhRespInfo == nil ||
			ebhRespInfo.Timestamp < requestTimestamp || !ebhRespInfo.Data.IsHeaderHashFinalized {
			peerGroup = append(peerGroup[:i], peerGroup[i+1:]...)
			i--
			continue
		}
		peerAllHashesBlank := true
		for _, blockHashesBySlotNumber := range ebhRespInfo.Data.EpochBlockHashesMetaData.BlockHashesBySlotNumber {
			slotNumber := blockHashesBySlotNumber.SlotNumber
			if len(blockHashesBySlotNumber.HeaderHashes) == 0 {
				continue
			}
			peerAllHashesBlank = false
			// Ignore genesis hash
			if slotNumber == 0 {
				continue
			}

			// Ignore finalized slotNumber
			if slotNumber < minimumStartSlotNumber {
				continue
			}

			// Ignore if peer sent slotNumber info of a different epoch
			if peer.GetEpochToBeRequested() != slotNumber/blocksPerEpoch {
				// TODO: Possibly peer must be banned
				continue
			}
			hashesWithPeers, ok := hashesWithPeerInfoBySlotNumber[slotNumber]
			if !ok {
				hashesWithPeers = &HashesAndPeerInfo{
					slotNumber:      slotNumber,
					peerByBlockHash: make(map[string][]*Peer),
				}
				hashesWithPeerInfoBySlotNumber[slotNumber] = hashesWithPeers
			}
			for _, headerHash := range blockHashesBySlotNumber.HeaderHashes {
				strHeaderHash := misc.BytesToHexStr(headerHash)
				data, ok := hashesWithPeers.peerByBlockHash[strHeaderHash]
				if !ok {
					data = make([]*Peer, 0)
					hashesWithPeers.peerByBlockHash[strHeaderHash] = data
				}
				hashesWithPeers.peerByBlockHash[strHeaderHash] = append(data, peer)
			}
		}

		if peerAllHashesBlank {
			peer.IncreaseEpochToBeRequested()
			peerGroup = append(peerGroup[:i], peerGroup[i+1:]...)
			i--
		}
	}

	if len(peerGroup) == 0 {
		log.Info("Downloading finished")
		d.isSyncing = false
		return
	}
	if len(hashesWithPeerInfoBySlotNumber) == 0 {
		for _, peer := range peerGroup {
			peer.IncreaseEpochToBeRequested()
		}
		log.Info("No hashes info response received by peers")
		d.isSyncingFinished(true, common.Hash{}, nil)
		return
	}

	targetSlotNumbers := make([]*HashesAndPeerInfo, 0, len(hashesWithPeerInfoBySlotNumber))
	for k := range hashesWithPeerInfoBySlotNumber {
		targetSlotNumbers = append(targetSlotNumbers, hashesWithPeerInfoBySlotNumber[k])
	}
	sort.Slice(targetSlotNumbers,
		func(i, j int) bool {
			return targetSlotNumbers[i].slotNumber < targetSlotNumbers[j].slotNumber
		})
	d.requestTracker = NewRequestTracker()
	d.ignorePeers = make(map[string]int)

	var lastBlockHeaderToDownload common.Hash
	var binData []byte
	var err error

	for headerHash := range targetSlotNumbers[len(targetSlotNumbers)-1].peerByBlockHash {
		binData, err = misc.HexStrToBytes(headerHash)
		copy(lastBlockHeaderToDownload[:], binData)
		if err != nil {
			d.isSyncing = false
			log.Error("Error decoding target header hash ", err.Error())
			return
		}
	}

	go d.Consumer(lastBlockHeaderToDownload, peerGroup)
	go d.BlockDownloader(targetSlotNumbers, lastBlockHeaderToDownload, peerGroup)
}
