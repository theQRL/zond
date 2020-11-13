package consensus

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/address"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/p2p"
	"github.com/theQRL/zond/state"
	"reflect"
	"sync"
	"time"
)

type POS struct {
	config *config.Config
	srv    *p2p.Server
	chain  *chain.Chain
	db     *db.DB

	// Dilithium PK is the key of this map
	validators map[string]*dilithium.Dilithium

	exit                        chan struct{}
	loopWG                      sync.WaitGroup
	blockReceivedForAttestation chan *block.Block
	attestationReceivedForBlock chan *transactions.Attest

	blockBeingAttested *block.Block // Block being proposed by this node and
	// awaiting for the attestations
	attestors [][]byte
	attestations []*transactions.Attest
}

func (p *POS) Start()  {
	/* TODO
		1. Load Dilithium Keys from file based in config
		2. Check if Stake Enabled
		3.
	 */

	/* Scenarios
	1.
	Block coming to Chain. After adding to block, Chain calls POS,
	with next slotleader dilithium address
	POS checks if the slotleader private key is present. If present
	then load the private key, make the blog, broadcast the unsigned,
	block for attestation

	2.
	Block was not received within a specific block timing.
	POS is called with next slot leader due to timeout,
	then check if next slotleader key is present

	 */

	/*
	POS will call itself after every

	1. sleep for next slot time - current time
	2. On Timer, check if slot leader is one of the address from wallet
	3. If yes, then fetch last block from Chain, create a new block,
	   broadcast it to server for attestation
	4. Keep adding attestation to the block, as soon as a threshold,
	   sign the block and broadcast it.
	5.

	 */
}

func (p *POS) TimeRemainingForNextAction() time.Duration {
	if p.blockBeingAttested == nil {
		// Wait time to propose the block
		currentTime := uint64(time.Now().Unix())
		genesisTimestamp := p.config.Dev.Genesis.GenesisTimestamp
		blockTiming := p.config.Dev.BlockTime

		currentSlot := (currentTime - genesisTimestamp) / blockTiming
		nextSlotTime := genesisTimestamp + (currentSlot + 1) * blockTiming

		timeRemainingForNextSlot := nextSlotTime - currentTime
		return time.Duration(timeRemainingForNextSlot) * time.Second
	} else {
		// If block has been proposed by this node,
		// then max wait time to collect attestor txn
		// before broadcasting the block
		currentTime := uint64(time.Now().Unix())
		slotNumber := p.blockBeingAttested.SlotNumber()
		genesisTimestamp := p.config.Dev.Genesis.GenesisTimestamp
		blockTiming := p.config.Dev.BlockTime
		// TODO: move this 45 seconds into config
		return time.Duration(
			(genesisTimestamp + slotNumber * blockTiming + 45) - currentTime) * time.Second
	}
}

func (p *POS) GetCurrentSlot() uint64 {
	currentTime := uint64(time.Now().Unix())
	genesisTimestamp := p.config.Dev.Genesis.GenesisTimestamp
	blockTiming := p.config.Dev.BlockTime

	return (currentTime - genesisTimestamp) / blockTiming
}

func (p *POS) Run() {
	p.loopWG.Add(1)
	defer p.loopWG.Done()

running:
	for {
		select {
		case <-p.exit:
			log.Info("Shutting Down POS")
			break running
		case <-time.After(p.TimeRemainingForNextAction()):
			if !p.config.User.Stake.EnableStaking {
				continue
			}
			if p.blockBeingAttested == nil {
				lastBlock := p.chain.GetLastBlock()

				slotNumber := p.GetCurrentSlot()
				slotLeader, err := p.chain.GetSlotLeaderDilithiumPKBySlotNumber(slotNumber,
					lastBlock.HeaderHash(), lastBlock.SlotNumber())

				if err != nil {
					log.Error("Error getting SlotLeader Dilithium PK By Slot Number ", err.Error())
					continue
				}
				proposerD, ok := p.validators[misc.Bin2HStr(slotLeader)]
				if !ok {
					continue
				}
				mainChainMetaData, err := metadata.GetMainChainMetaData(p.db)
				if err != nil {
					log.Error("Failed to Get MainChainMetaData ", err)
					continue
				}
				coinBaseState, err := address.GetAddressState(p.db, config.GetDevConfig().Genesis.CoinBaseAddress,
					lastBlock.HeaderHash(), mainChainMetaData.FinalizedBlockHeaderHash())
				if err != nil {
					log.Error("Error getting CoinBase Address state ", err.Error())
					continue
				}
				log.Info("Minting Block #", slotNumber)
				b := block.NewBlock(0, ntp.GetNTP().Time(), proposerD.PK(), slotNumber,
					lastBlock.HeaderHash(), nil, nil, coinBaseState.Nonce())

				header := b.Header()
				attestors, err := p.chain.GetAttestorsBySlotNumber(header.SlotNumber,
					header.ParentHeaderHash, lastBlock.SlotNumber())
				if err != nil {
					log.Error("Error while getting Attestors by Slot Number ", err.Error())
					continue
				}
				p.blockBeingAttested = b
				p.attestors = attestors
				p.attestations = make([]*transactions.Attest, 0)
				for _, dilithiumPK := range attestors {
					strDilithiumPK := misc.Bin2HStr(dilithiumPK)
					d, ok := p.validators[strDilithiumPK]
					if !ok {
						continue
					}
					attestTx, err := b.Attest(0, d)
					if err != nil {
						log.Error("Error while Attesting ", err.Error())
					}
					p.attestations = append(p.attestations, attestTx)
				}

				// In case of all attestors for this slot belongs to same node
				// then we already have all attestors and we should broadcast
				// the block
				if len(p.attestations) == len(attestors) {
					p.blockBeingAttested.SignByProposer(proposerD)
					b = p.blockBeingAttested
					p.blockBeingAttested = nil
					p.attestations = make([]*transactions.Attest, 0)

					if !p.chain.AddBlock(b) {
						log.Error("Error adding block proposed by this Node")
						continue
					}
					p.srv.BroadcastBlock(b)
					continue
				}

				/*
				1. If any attestation remaining,
				then Broadcast Unsigned block via server for attestation
				2. Optimization needed PartialBlockSigningHash is called two times
				 */
				partialBlockSigningHash := b.PartialBlockSigningHash()
				log.Info("Broadcasting Block #", slotNumber, " for attestation")
				p.srv.BroadcastBlockForAttestation(b, proposerD.Sign(partialBlockSigningHash))

			} else {
				// TODO:
				// Check if block has sufficient attestation
				// If yes then add block to the chain and
				// broadcast the block
				if len(p.blockBeingAttested.ProtocolTransactions()) > 1 {
					log.Info("Number of Attestations Received ",
						len(p.blockBeingAttested.ProtocolTransactions()) - 1,
						" for Block #", p.blockBeingAttested.SlotNumber())
					p.chain.AddBlock(p.blockBeingAttested)
					p.srv.BroadcastBlock(p.blockBeingAttested)
				} else {
					log.Info("Insufficient attestation for Block #",
						p.blockBeingAttested.SlotNumber())
				}

				p.blockBeingAttested = nil
				p.attestations = make([]*transactions.Attest, 0)
			}
		case b := <- p.blockReceivedForAttestation:
			if !p.config.User.Stake.EnableStaking {
				continue
			}
			/*
			This case happens, when the block is proposed by some outside node.
			 */
			/*
				TODO:
				Check if block slot number must be equal to the current expected slot number
				check if the block slot leader is correct
				Don't accept Block for attestation after certain threshold
			*/

			if !reflect.DeepEqual(b.ParentHeaderHash(), p.chain.GetLastBlock().HeaderHash()) {
				continue
			}

			if p.blockBeingAttested != nil && p.blockBeingAttested.SlotNumber() == b.SlotNumber() {
				continue
			}
			header := b.Header()
			parentBlock, err := p.chain.GetBlock(header.ParentHeaderHash)
			if err != nil {
				log.Error("Failed to Get Parent Block ", err.Error())
				continue
			}
			attestors, err := p.chain.GetAttestorsBySlotNumber(header.SlotNumber,
				header.ParentHeaderHash, parentBlock.SlotNumber())
			if err != nil {
				log.Error("Error while getting Attestors by Slot Number ", err.Error())
				continue
			}
			p.blockBeingAttested = b
			log.Info("Block #", b.SlotNumber(), " received for attestation")
			partialBlockSigningHash := b.PartialBlockSigningHash()
			blockProposer := b.ProtocolTransactions()[0].GetPk()
			for _, dilithiumPK := range attestors {
				strDilithiumPK := misc.Bin2HStr(dilithiumPK)
				d, ok := p.validators[strDilithiumPK]
				if !ok {
					continue
				}
				attestTx, err := b.Attest(0, d)
				if err != nil {
					log.Error("Error while Attesting ", err.Error())
				}
				p.srv.BroadcastAttestationTransaction(attestTx,
					b.SlotNumber(),
					blockProposer,
					b.ParentHeaderHash(),
					partialBlockSigningHash)
			}
		case tx := <- p.attestationReceivedForBlock:
			if !p.config.User.Stake.EnableStaking {
				continue
			}
			if p.blockBeingAttested == nil {
				continue
			}
			/*
			When the attestation is received, check if the attest transaction is for the block
			proposed by this node.
			 */

			// Validate Txn
			epochMetaData, err := metadata.GetEpochMetaData(p.db, p.blockBeingAttested.SlotNumber(),
				p.blockBeingAttested.ParentHeaderHash())
			if err != nil{
				log.Error("Error getting epochMetaData")
				continue
			}
			sc, err := state.NewStateContext(p.db, p.blockBeingAttested.SlotNumber(),
				p.blockBeingAttested.ProtocolTransactions()[0].Pk,
				p.blockBeingAttested.ParentHeaderHash(),
				p.blockBeingAttested.HeaderHash(),
				p.blockBeingAttested.PartialBlockSigningHash(),
				nil,
				epochMetaData)
			if err != nil {
				log.Error("Error creating NewStateContext")
				continue
			}
			if err := tx.SetAffectedAddress(sc); err != nil {
				log.Error("Failed to set Affected Address")
				continue
			}
			if !tx.Validate(sc) {
				log.Warn("Attestor transaction validation failed")
				continue
			}

			// Ignore duplicate Attest Transaction
			isDuplicate := false
			for _, oldTx := range p.attestations {
				if reflect.DeepEqual(oldTx.PK(), tx.PK()) {
					isDuplicate = true
					break
				}
			}
			if isDuplicate {
				continue
			}

			p.attestations = append(p.attestations, tx)

			// Add received attestation into block
			p.blockBeingAttested.AddAttestTx(tx)

			// Check if all attestations has been received, if yes, then broadcast block
			//if len(p.attestations) == len(p.attestors) {
			//	slotLeader := p.blockBeingAttested.ProtocolTransactions()[0].Pk
			//	proposerD, ok := p.validators[misc.Bin2HStr(slotLeader)]
			//	if !ok {
			//		continue
			//	}
			//	p.blockBeingAttested.SignByProposer(proposerD)
			//	p.srv.BroadcastBlock(p.blockBeingAttested)
			//	p.blockBeingAttested = nil
			//	p.attestations = make([]*transactions.Attest, 0)
			//}
		}
	}
}

func (p *POS) Stop() {
	close(p.exit)
	p.loopWG.Wait()

	return
}

func NewPOS(srv *p2p.Server, chain *chain.Chain, db *db.DB) *POS {
	pos := &POS {
		config: config.GetConfig(),
		srv: srv,
		chain: chain,
		db: db,
		validators: make(map[string] *dilithium.Dilithium),
		exit: make(chan struct{}),

		blockReceivedForAttestation: srv.GetBlockReceivedForAttestation(),
		attestationReceivedForBlock: srv.GetAttestationReceivedForBlock(),
	}

	dk := keys.NewDilithiumKeys(pos.config.User.Stake.DilithiumKeysFileName)
	for _, group := range dk.GetDilithiumGroup() {
		for _, dilithiumInfo := range group.DilithiumInfo {
			strPK := dilithiumInfo.PK
			pk := misc.HStr2Bin(dilithiumInfo.PK)
			sk := misc.HStr2Bin(dilithiumInfo.SK)
			pos.validators[strPK] = dilithium.RecoverDilithium(pk, sk)
		}
	}

	return pos
}
