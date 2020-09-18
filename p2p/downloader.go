package p2p

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/protos"
	"sort"
	"sync"
	"time"
)

const MaxRequest = 40 // Size has to be calculated based on maximum possible values on Queue

type BlockAndPeer struct {
	block *block.Block
	peer *Peer
}

type HashesAndPeerInfo struct {
	slotNumber      uint64
	peerByBlockHash map[string][]*Peer
}

type RequestTracker struct {
	lock sync.Mutex

	requests map[string]*Peer
	sequence []string  // Sequence of Header hashes in which request was made
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

type Downloader struct {
	lock sync.Mutex

	isSyncing  bool
	chain      *chain.Chain
	ntp        ntp.NTPInterface

	blockAndPeerChannel chan *BlockAndPeer

	wg                        sync.WaitGroup
	exitDownloadMonitor       chan struct{}
	peersList                 map[string]*Peer
	requestTracker            *RequestTracker // Maintains block header hash requested by the peer
	peerSelectionCount        int
	slotNumberProcessed       chan uint64
	done                      chan struct{}
	consumerRunning           bool
	blockDownloaderRunning    bool
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

//func (d *Downloader) AddPeer(p *Peer) {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	if _, ok := d.peersList[p.ID()]; ok {
//		return
//	}
//
//	d.peersList[p.ID()] = p
//}
//
//func (d *Downloader) RemovePeer(p *Peer) bool {
//	d.lock.Lock()
//	defer d.lock.Unlock()
//
//	if _, ok := d.peersList[p.ID()]; !ok {
//		return false
//	}
//
//	delete(d.peersList, p.ID())
//	return true
//}
//
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

func (d *Downloader) Consumer() {
	d.consumerRunning = true
	defer func () {
		d.consumerRunning = false
	}()
	pendingBlocks := make(map[string]*block.Block)
	for {
		select {
		case blockAndPeer := <-d.blockAndPeerChannel:
			b := blockAndPeer.block
			// Ensure if the block received is from the same Peer from which it was requested
			strHeaderHash := misc.Bin2HStr(blockAndPeer.block.HeaderHash())
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
					"#", b.SlotNumber(),
					"headerhash", misc.Bin2HStr(b.HeaderHash()))
				delete(pendingBlocks, strHeaderHash)
				d.requestTracker.RemoveFirstElementFromSequence()
				d.requestTracker.RemoveRequestKey(strHeaderHash)

				if !d.chain.AddBlock(b) {
					log.Warn("Failed To Add Block")
					break
				}

				log.Info("Block Added",
					"BlockNumber", b.SlotNumber(),
					"Headerhash", misc.Bin2HStr(b.HeaderHash()))

				d.slotNumberProcessed <- b.SlotNumber()
			}

			if d.isSyncingFinished(false) {
				log.Info("Block Download Syncing Finished")
				return
			}
		case <-time.After(30*time.Second):
			log.Info("[Consumer Timeout] Finishing downloading")
			d.isSyncingFinished(true)
			return
		case <-d.done:
			d.isSyncing = false
			return
		}
	}
}

func (d *Downloader) RequestForBlock(hashesPeerInfo *HashesAndPeerInfo, numberOfRequests int) (int, error) {
	/*
	Request for child header hash from all peers
	 */
	log.Info("Requesting For Block")

	for headerHash := range hashesPeerInfo.peerByBlockHash {
		peerLen := len(hashesPeerInfo.peerByBlockHash[headerHash])
		for ; peerLen > 0; {
			selectedPeer := (hashesPeerInfo.slotNumber) % uint64(peerLen)
			peer := hashesPeerInfo.peerByBlockHash[headerHash][selectedPeer]
			hashesPeerInfo.peerByBlockHash[headerHash] = hashesPeerInfo.peerByBlockHash[headerHash][1:]
			peerLen--
			if peer == nil || peer.disconnected {
				continue
			}
			err := peer.SendFetchBlock(misc.HStr2Bin(headerHash))
			if err != nil {
				log.Error("Error fetching block from peer ", peer.ID())
				continue
			}
			d.requestTracker.AddPeerRequest(headerHash, peer)
			numberOfRequests++
			break
		}
	}
	return numberOfRequests, nil
}

func (d *Downloader) BlockDownloader(targetSlotNumbers []*HashesAndPeerInfo) {
	d.blockDownloaderRunning = true
	defer func () {
		d.blockDownloaderRunning = false
	}()

	log.Info("Block Downloader Started")

	maxRequestedIndex := 0
	numberOfRequests, err := d.RequestForBlock(targetSlotNumbers[maxRequestedIndex], 0)
	if err != nil {
		log.Error("Error while Requesting for block",
			"Block #", targetSlotNumbers[maxRequestedIndex].slotNumber,
			"Error", err.Error())
	}
	targetSlotNumbersLen := len(targetSlotNumbers)
	for {
		select {
		case blockNumber := <-d.slotNumberProcessed:
			numberOfRequests -= 1
			for numberOfRequests < MaxRequest - 10 && maxRequestedIndex + 1 < targetSlotNumbersLen {
				numberOfRequests, err = d.RequestForBlock(targetSlotNumbers[maxRequestedIndex + 1],
					numberOfRequests)
				if err != nil {
					log.Error("Error while Requesting for block",
						"Block #", blockNumber,
						"Error", err.Error())
					continue
				}
				maxRequestedIndex += 1
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

func (d *Downloader) isSyncingFinished(forceFinish bool) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.isSyncing {
		return true
	}

	if forceFinish {
		log.Info("Syncing FINISHED")
		return true
	}
	return false
}

func NewDownloader(c *chain.Chain) (d *Downloader) {
	d = &Downloader {
		isSyncing: false,
		chain: c,
		ntp: ntp.GetNTP(),

		peersList: make(map[string]*Peer),
		peerSelectionCount: 0,
		slotNumberProcessed: make(chan uint64, MaxRequest),
		blockAndPeerChannel: make(chan *BlockAndPeer, MaxRequest*2),
		done: make(chan struct{}),
	}
	return
}

func (d *Downloader) Exit() {
	log.Debug("Shutting Down Downloader")
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
		case <- time.After(30 * time.Second):
			// Ignore if Consumer or BlockDownloader already running.
			if d.isDownloaderRunning() {
				continue
			}
			if d.isSyncing {
				continue
			}
			maxPeerGroupLen := 0
			var targetHeaderHash []byte
			peerGroupByHeaderHash := make(map[string] []*Peer)
			for _, p := range d.peersList {
				// check if peer already exists
				if _, ok := peerGroupByHeaderHash[p.ID()]; ok {
					continue
				}
				peerChainState := p.ChainState()
				if peerChainState == nil {
					continue
				}

				// If block not found, then syncing is required
				if _, err := d.chain.GetBlock(peerChainState.HeaderHash); err != nil {
					peerGroup := peerGroupByHeaderHash[misc.Bin2HStr(peerChainState.HeaderHash)]
					peerGroup = append(peerGroup, p)
					if len(peerGroup) > maxPeerGroupLen {
						maxPeerGroupLen = len(peerGroup)
						targetHeaderHash = peerChainState.HeaderHash
					}
				}
			}

			if len(peerGroupByHeaderHash) == 0 {
				continue
			}

			/*
				Select group with highest number of peers and
				consider them as target peer

			*/
			peerGroup := peerGroupByHeaderHash[misc.Bin2HStr(targetHeaderHash)]
			log.Info("===================== downloading ====================")
			d.Initialize(peerGroup, targetHeaderHash)
		case <-d.exitDownloadMonitor:
			return
		}
	}
}

func (d *Downloader) Initialize(peerGroup []*Peer, targetHeaderHash []byte) {
	/*
	Syncing start after finalized block
	1. Request for all child header hashes from all peers of an epoch.
	2. Assign header hashes with Peer
	3. Request for block from randomly selected peer
	 */
	log.Info("Initializing Downloader")
	requestTimestamp := d.ntp.Time()
	for _, peer := range peerGroup {
		peer.SendEBHReq(targetHeaderHash)
	}
	d.isSyncing = true
	d.done = make(chan struct{})

	// Delay 15 seconds to receive hashes from peers
	select {
	case <- time.After(15 * time.Second):
	}

	hashesWithPeerInfoBySlotNumber := make(map[uint64] *HashesAndPeerInfo)
	for i := 0; i < len(peerGroup); i++ {
		peer := peerGroup[i]
		ebhRespInfo := peer.ebhRespInfo
		if peer.disconnected || ebhRespInfo == nil || ebhRespInfo.Timestamp < requestTimestamp {
			peerGroup = append(peerGroup[:i], peerGroup[i+1:]...)
			i--
			continue
		}
		for _, blockHashesBySlotNumber := range ebhRespInfo.Data.BlockHashesBySlotNumber {
			slotNumber := blockHashesBySlotNumber.SlotNumber
			hashesWithPeers, ok := hashesWithPeerInfoBySlotNumber[slotNumber]
			if !ok {
				hashesWithPeers = &HashesAndPeerInfo {
					slotNumber: slotNumber,
					peerByBlockHash: make(map[string] []*Peer),
				}
				hashesWithPeerInfoBySlotNumber[slotNumber] = hashesWithPeers
			}
			for _, headerHash := range blockHashesBySlotNumber.HeaderHashes {
				strHeaderHash := misc.Bin2HStr(headerHash)
				data, ok := hashesWithPeers.peerByBlockHash[strHeaderHash]
				if !ok {
					data = make([]*Peer, 0)
					hashesWithPeers.peerByBlockHash[strHeaderHash] = data
				}
				hashesWithPeers.peerByBlockHash[strHeaderHash] = append(data, peer)
			}
		}
	}

	targetSlotNumbers := make([]*HashesAndPeerInfo, 0, len(hashesWithPeerInfoBySlotNumber))
	for k := range hashesWithPeerInfoBySlotNumber {
		targetSlotNumbers = append(targetSlotNumbers, hashesWithPeerInfoBySlotNumber[k])
	}
	sort.Slice(targetSlotNumbers,
		func(i, j int) bool {
			return targetSlotNumbers[i].slotNumber < targetSlotNumbers[j].slotNumber
		})
	d.requestTracker = &RequestTracker{}
	go d.Consumer()
	go d.BlockDownloader(targetSlotNumbers)
}
