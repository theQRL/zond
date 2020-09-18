package chain

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/block/genesis"
	"github.com/theQRL/zond/chain/transactions/pool"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"sort"
	"sync"
)

type Chain struct {
	lock sync.Mutex

	config *config.Config

	state *state.State

	txPool *pool.TransactionPool

	lastBlock *block.Block
}

func (c *Chain) GetTransactionPool() *pool.TransactionPool {
	return c.txPool
}

func (c *Chain) GetLastBlock() *block.Block {
	return c.lastBlock
}

func (c *Chain) Height() uint64 {
	return c.lastBlock.SlotNumber()
}

func (c *Chain) Load() error {
	db := c.state.DB()
	mainChainMetaData, err := metadata.GetMainChainMetaData(db)
	if err != nil {
		err := genesis.ProcessGenesisBlock(db)
		if err != nil {
			log.Error("Failed to Process Genesis Block")
			return err
		}
		mainChainMetaData, err = metadata.GetMainChainMetaData(db)
		if err != nil {
			log.Error("Failed to Load MainChainMetaData")
			return err
		}
	}
	if mainChainMetaData == nil {
		return errors.New("MainChainMetaData cannot be nil")
	}

	lastBlock, err := block.GetBlock(db, mainChainMetaData.LastBlockHeaderHash())
	if err != nil {
		return err
	}
	if lastBlock == nil {
		return errors.New("LastBlock cannot be nil")
	}

	c.lastBlock = lastBlock

	return nil
}

func (c *Chain) GetSlotLeaderDilithiumPKBySlotNumber(slotNumber uint64,
	parentHeaderHash []byte, parentSlotNumber uint64) ([]byte, error) {
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(),
		slotNumber, parentHeaderHash, parentSlotNumber)
	if err != nil {
		return nil, err
	}

	slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber % c.config.Dev.BlocksPerEpoch].SlotLeader
	return epochMetaData.Validators()[slotLeaderIndex], nil
}

func (c *Chain) GetAttestorsBySlotNumber(slotNumber uint64,
	parentHeaderHash []byte, parentSlotNumber uint64) ([][]byte, error) {
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(), slotNumber,
		parentHeaderHash, parentSlotNumber)
	if err != nil {
		return nil, err
	}
	attestorsIndex := epochMetaData.SlotInfo()[slotNumber % c.config.Dev.BlocksPerEpoch].Attestors
	validators := epochMetaData.Validators()

	var attestors [][]byte
	for _, attestorIndex := range attestorsIndex {
		attestors = append(attestors, validators[attestorIndex])
	}

	return attestors, nil
}

func (c *Chain) GetBlockMetaData(headerHash []byte) (*metadata.BlockMetaData, error) {
	return metadata.GetBlockMetaData(c.state.DB(), headerHash)
}

func (c *Chain) GetBlock(headerHash []byte) (*block.Block, error) {
	return block.GetBlock(c.state.DB(), headerHash)
}

func (c *Chain) GetEpochHeaderHashes(headerHash []byte) ([]*protos.BlockHashesBySlotNumber, error) {
	hashesBySlotNumber := make(map[uint64]*protos.BlockHashesBySlotNumber)
	b, err := c.GetBlock(headerHash)
	if err != nil {
		return nil, err
	}

	blockMetaData, err := c.GetBlockMetaData(headerHash)
	if err != nil {
		return nil, err
	}

	var childHeaderHashes [][]byte

	/*
	Initializing with next child header hashes, which comes just
	after the finalized block header hash.
	We assume if the current block header hash is finalized, then it
	is the last block of the epoch. In such a case, we expect
	the child header hash must be from epoch higher than the finalized
	block.
	In normal cases, child header hash will be from the 1 epoch ahead
	from the finalized block.
	In rare cases, it may be possible that child header hash is more than
	1 epoch ahead from the finalized block, given no slot leader mint
	any block for the 1 epoch ahead from the finalized block.
	 */
	if blockMetaData.FinalizedChildHeaderHash() == nil {
		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
			childBlock, err := c.GetBlock(childHeaderHash)
			if err != nil {
				log.Error("Error getting child block")
				return nil, err
			}

			// Ignore this condition if the finalized block is a genesis block
			if childBlock.Epoch() == b.Epoch() && b.SlotNumber() != 0 {
				continue
			}
			data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
			if !ok {
				data = &protos.BlockHashesBySlotNumber {
					SlotNumber: childBlock.SlotNumber(),
				}
				hashesBySlotNumber[childBlock.SlotNumber()] = data
			}
			data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
			childHeaderHashes = append(childHeaderHashes, childBlock.HeaderHash())
		}
	} else {
		childBlock, err := c.GetBlock(blockMetaData.FinalizedChildHeaderHash())
		if err != nil {
			log.Error("Unable to GetBlock by FinalizedChildHeaderHash")
			return nil, err
		}
		data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
		if !ok {
			data = &protos.BlockHashesBySlotNumber{
				SlotNumber: childBlock.SlotNumber(),
			}
			hashesBySlotNumber[childBlock.SlotNumber()] = data
		}
		data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
		childHeaderHashes = append(childHeaderHashes, childBlock.HeaderHash())
	}

	for ;len(childHeaderHashes) > 0; {
		headerHash := childHeaderHashes[0]
		childHeaderHashes = childHeaderHashes[1:]

		b, err := c.GetBlock(headerHash)
		if err != nil {
			log.Error("Failed to GetBlock ", misc.Bin2HStr(headerHash))
			return nil, err
		}

		blockMetaData, err := c.GetBlockMetaData(headerHash)
		if err != nil {
			log.Error("Failed to GetBlockMetaData ", misc.Bin2HStr(headerHash))
			return nil, err
		}

		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
			childBlock, err := c.GetBlock(childHeaderHash)
			if err != nil {
				log.Error("Error getting child block ", misc.Bin2HStr(childHeaderHash))
				return nil, err
			}

			if childBlock.Epoch() != b.Epoch() {
				continue
			}
			data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
			if !ok {
				data = &protos.BlockHashesBySlotNumber {
					SlotNumber: childBlock.SlotNumber(),
				}
				hashesBySlotNumber[childBlock.SlotNumber()] = data
			}
			data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
			childHeaderHashes = append(childHeaderHashes, childHeaderHash)
		}
	}

	keys := make([]uint64, 0, len(hashesBySlotNumber))
	for k := range hashesBySlotNumber {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var protoHashesBySlotNumbers []*protos.BlockHashesBySlotNumber
	for _, key := range keys {
		data, _ := hashesBySlotNumber[key]
		protoHashesBySlotNumbers = append(protoHashesBySlotNumbers, data)
	}

	return protoHashesBySlotNumbers, nil
}

func (c *Chain) AddBlock(b *block.Block) bool {
	/* TODO: Revise Block Validation */
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("[AddBlock] Failed to Get MainChainMetaData ", err.Error())
		return false
	}
	err = b.Commit(c.state.DB(), mainChainMetaData.FinalizedBlockHeaderHash(), false)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to commit block #%d %s | Error %s", b.SlotNumber(),
			misc.Bin2HStr(b.HeaderHash()), err.Error()))
		return false
	}

	log.Info(fmt.Sprintf("Added Block #%d %s", b.SlotNumber(), misc.Bin2HStr(b.HeaderHash())))
	return true
}

func (c *Chain) GetStateContext() (*state.StateContext, error) {
	lastBlock := c.lastBlock

	epochMetaData, err := metadata.GetEpochMetaData(c.state.DB(),
		lastBlock.SlotNumber(), lastBlock.ParentHeaderHash())
	if err != nil {
		return nil, err
	}

	return state.NewStateContext(c.state.DB(), lastBlock.SlotNumber(), nil,
		lastBlock.ParentHeaderHash(), lastBlock.HeaderHash(),
		lastBlock.PartialBlockSigningHash(), lastBlock.BlockSigningHash(),
		epochMetaData)
}

func NewChain(s *state.State) *Chain {
	return &Chain {
		config: config.GetConfig(),
		state: s,
		txPool: pool.CreateTransactionPool(),
	}
}
