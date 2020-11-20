package chain

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/address"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/block/genesis"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/chain/transactions/pool"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"math/big"
	"reflect"
	"sync"
)

type Chain struct {
	lock sync.Mutex

	config *config.Config

	state *state.State

	txPool *pool.TransactionPool

	lastBlock *block.Block
}

func (c *Chain) GetAddressState(addr []byte) (*address.AddressState, error) {
	finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		return nil, err
	}
	return address.GetAddressState(c.state.DB(), addr, c.lastBlock.HeaderHash(),
		finalizedHeaderHash)
}

func (c *Chain) GetMaxPossibleSlotNumber() uint64 {
	d := config.GetDevConfig()
	currentTimestamp := ntp.GetNTP().Time()
	genesisTimestamp := d.Genesis.GenesisTimestamp

	return (currentTimestamp - genesisTimestamp) / d.BlockTime
}

func (c *Chain) GetTransactionPool() *pool.TransactionPool {
	return c.txPool
}

func (c *Chain) GetLastBlock() *block.Block {
	return c.lastBlock
}

func (c *Chain) GetTotalStakeAmount() (*big.Int, error) {
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("[GetTotalStakeAmount] Failed to get MainChainMetaData ", err.Error())
		return nil, err
	}
	if mainChainMetaData.LastBlockHeaderHash() == nil {
		log.Error("[GetTotalStakeAmount] MainChainMetaData LastBlockHeaderHash is nil")
		return nil, err
	}
	lastBlockMetaData, err := c.GetBlockMetaData(mainChainMetaData.LastBlockHeaderHash())
	if err != nil {
		log.Error("[GetTotalStakeAmount] Failed to load LastBlockMetaData ", err.Error())
		return nil, err
	}
	totalStakeAmount := big.NewInt(0)
	err = totalStakeAmount.UnmarshalText(lastBlockMetaData.TotalStakeAmount())
	if err != nil {
		log.Error("[GetTotalStakeAmount] Failed to unmarshal TotalStakeAmount ", err.Error())
		return nil, err
	}
	return totalStakeAmount, nil
}

func (c *Chain) GetFinalizedHeaderHash() ([]byte, error) {
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		return nil, err
	}
	return mainChainMetaData.FinalizedBlockHeaderHash(), nil
}

func (c *Chain) GetStartingNonFinalizedEpoch() (uint64, error) {
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		return 0, err
	}
	// Special condition for epoch 0, to start syncing from epoch 0
	if mainChainMetaData.FinalizedBlockSlotNumber() < config.GetDevConfig().BlocksPerEpoch - 1 {
		return 0, nil
	}
	finalizedEpoch := mainChainMetaData.FinalizedBlockSlotNumber() / config.GetDevConfig().BlocksPerEpoch

	return finalizedEpoch + 1, nil

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
		log.Error("Failed to load last block for ",
			misc.Bin2HStr(mainChainMetaData.LastBlockHeaderHash()))
		return err
	}
	if lastBlock == nil {
		return errors.New("LastBlock cannot be nil")
	}

	c.lastBlock = lastBlock
	log.Info(fmt.Sprintf("Current Block Slot Number %d HeaderHash %s",
		c.lastBlock.SlotNumber(), misc.Bin2HStr(c.lastBlock.HeaderHash())))
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

//func (c *Chain) GetEpochHeaderHashes(headerHash []byte) ([]*protos.BlockHashesBySlotNumber, error) {
//	hashesBySlotNumber := make(map[uint64]*protos.BlockHashesBySlotNumber)
//	b, err := c.GetBlock(headerHash)
//	if err != nil {
//		return nil, err
//	}
//
//	blockMetaData, err := c.GetBlockMetaData(headerHash)
//	if err != nil {
//		return nil, err
//	}
//
//	var childHeaderHashes [][]byte
//
//	/*
//	Initializing with next child header hashes, which comes just
//	after the finalized block header hash.
//	We assume if the current block header hash is finalized, then it
//	is the last block of the epoch. In such a case, we expect
//	the child header hash must be from epoch higher than the finalized
//	block.
//	In normal cases, child header hash will be maximum 1 epoch ahead
//	from the finalized block.
//	In rare cases, it may be possible that child header hash is more than
//	1 epoch ahead from the finalized block, given no slot leader mint
//	any block for the 1 epoch ahead from the finalized block.
//	 */
//	if blockMetaData.FinalizedChildHeaderHash() == nil {
//		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
//			childBlock, err := c.GetBlock(childHeaderHash)
//			if err != nil {
//				log.Error("Error getting child block")
//				return nil, err
//			}
//
//			// Ignore this condition if the finalized block is a genesis block
//			if childBlock.Epoch() == b.Epoch() && b.SlotNumber() != 0 {
//				continue
//			}
//			data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
//			if !ok {
//				data = &protos.BlockHashesBySlotNumber {
//					SlotNumber: childBlock.SlotNumber(),
//				}
//				hashesBySlotNumber[childBlock.SlotNumber()] = data
//			}
//			data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
//			childHeaderHashes = append(childHeaderHashes, childBlock.HeaderHash())
//		}
//	} else {
//		childBlock, err := c.GetBlock(blockMetaData.FinalizedChildHeaderHash())
//		if err != nil {
//			log.Error("Unable to GetBlock by FinalizedChildHeaderHash")
//			return nil, err
//		}
//		data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
//		if !ok {
//			data = &protos.BlockHashesBySlotNumber{
//				SlotNumber: childBlock.SlotNumber(),
//			}
//			hashesBySlotNumber[childBlock.SlotNumber()] = data
//		}
//		data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
//		childHeaderHashes = append(childHeaderHashes, childBlock.HeaderHash())
//	}
//
//	for ;len(childHeaderHashes) > 0; {
//		headerHash := childHeaderHashes[0]
//		childHeaderHashes = childHeaderHashes[1:]
//
//		b, err := c.GetBlock(headerHash)
//		if err != nil {
//			log.Error("Failed to GetBlock ", misc.Bin2HStr(headerHash))
//			return nil, err
//		}
//
//		blockMetaData, err := c.GetBlockMetaData(headerHash)
//		if err != nil {
//			log.Error("Failed to GetBlockMetaData ", misc.Bin2HStr(headerHash))
//			return nil, err
//		}
//
//		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
//			childBlock, err := c.GetBlock(childHeaderHash)
//			if err != nil {
//				log.Error("Error getting child block ", misc.Bin2HStr(childHeaderHash))
//				return nil, err
//			}
//
//			if childBlock.Epoch() != b.Epoch() {
//				continue
//			}
//			data, ok := hashesBySlotNumber[childBlock.SlotNumber()]
//			if !ok {
//				data = &protos.BlockHashesBySlotNumber {
//					SlotNumber: childBlock.SlotNumber(),
//				}
//				hashesBySlotNumber[childBlock.SlotNumber()] = data
//			}
//			data.HeaderHashes = append(data.HeaderHashes, childBlock.HeaderHash())
//			childHeaderHashes = append(childHeaderHashes, childHeaderHash)
//		}
//	}
//
//	keys := make([]uint64, 0, len(hashesBySlotNumber))
//	for k := range hashesBySlotNumber {
//		keys = append(keys, k)
//	}
//	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
//
//	var protoHashesBySlotNumbers []*protos.BlockHashesBySlotNumber
//	for _, key := range keys {
//		data, _ := hashesBySlotNumber[key]
//		protoHashesBySlotNumbers = append(protoHashesBySlotNumbers, data)
//	}
//
//	return protoHashesBySlotNumbers, nil
//}

func (c *Chain) GetEpochHeaderHashes(epoch uint64) (*protos.EpochBlockHashesMetaData, error) {
	epochBlockHashes, err := metadata.GetEpochBlockHashes(c.state.DB(), epoch)
	if err != nil {
		return metadata.NewEpochBlockHashes(epoch).PBData(), nil
	}
	return epochBlockHashes.PBData(), nil
}

func (c *Chain) ValidateTransaction(tx *transactions.Transaction) error {
	sc, err := c.GetStateContext()
	if err != nil {
		return err
	}

	if err := tx.SetAffectedAddress(sc); err != nil {
		return err
	}
	if !tx.Validate(sc) {
		return errors.New("transaction validation failed")
	}
	return nil
}

func (c *Chain) AddBlock(b *block.Block) bool {
	/* TODO: Revise Block Validation */
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("[AddBlock] Failed to Get MainChainMetaData ", err.Error())
		return false
	}
	parentBlock, err := c.GetBlock(b.ParentHeaderHash())
	if err != nil {
		log.Error("[AddBlock] Failed to Get ParentBlock ", err.Error())
		return false
	}
	if parentBlock.SlotNumber() < mainChainMetaData.FinalizedBlockSlotNumber() {
		log.Error("[AddBlock] ParentBlock slot number is less than finalized block slot number")
		return false
	}

	if parentBlock.SlotNumber() == mainChainMetaData.FinalizedBlockSlotNumber() {
		parentHeaderHash := parentBlock.HeaderHash()
		if !reflect.DeepEqual(parentHeaderHash, mainChainMetaData.FinalizedBlockHeaderHash()) {
			log.Error("[AddBlock] ParentBlock is not the part of the finalized chain",
				" Expected hash ", misc.Bin2HStr(mainChainMetaData.FinalizedBlockHeaderHash()),
				" Found hash ", misc.Bin2HStr(parentHeaderHash))
			return false
		}
	}

	err = b.Commit(c.state.DB(), mainChainMetaData.FinalizedBlockHeaderHash(), false)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to commit block #%d %s | Error %s", b.SlotNumber(),
			misc.Bin2HStr(b.HeaderHash()), err.Error()))
		return false
	}

	// Reload MainChainMetaData from db
	mainChainMetaData, err = metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("Failed to Get MainChainMetaData ", err.Error())
		return true
	}

	if reflect.DeepEqual(mainChainMetaData.LastBlockHeaderHash(), b.HeaderHash()) {
		c.lastBlock = b
	} else if !reflect.DeepEqual(mainChainMetaData.LastBlockHeaderHash(),
		c.lastBlock.HeaderHash()) {
		lastBlock, err := c.GetBlock(mainChainMetaData.LastBlockHeaderHash())
		if err != nil {
			log.Error("Failed to Get Block for LastBlockHeaderHash ",
				misc.Bin2HStr(mainChainMetaData.LastBlockHeaderHash()))
			return true
		}
		c.lastBlock = lastBlock
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

	finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		return nil, err
	}

	return state.
		NewStateContext(c.state.DB(), lastBlock.SlotNumber(), nil,
		finalizedHeaderHash, lastBlock.ParentHeaderHash(), lastBlock.HeaderHash(),
		lastBlock.PartialBlockSigningHash(), lastBlock.BlockSigningHash(),
		epochMetaData)
}

func (c *Chain) GetStateContext2(slotNumber uint64, blockProposer []byte,
	parentHeaderHash []byte, partialBlockSigningHash []byte) (*state.StateContext, error) {
	lastBlock := c.lastBlock

	epochMetaData, err := metadata.GetEpochMetaData(c.state.DB(),
		lastBlock.SlotNumber(), lastBlock.ParentHeaderHash())
	if err != nil {
		return nil, err
	}

	finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		return nil, err
	}

	return state.NewStateContext(c.state.DB(), slotNumber, blockProposer,
		finalizedHeaderHash, parentHeaderHash, nil,
		partialBlockSigningHash, nil,
		epochMetaData)
}

func NewChain(s *state.State) *Chain {
	return &Chain {
		config: config.GetConfig(),
		state: s,
		txPool: pool.CreateTransactionPool(),
	}
}
