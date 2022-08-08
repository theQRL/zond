package chain

import (
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/block/genesis"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/core"
	"github.com/theQRL/zond/core/rawdb"
	state2 "github.com/theQRL/zond/core/state"
	"github.com/theQRL/zond/core/vm"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/params"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"github.com/theQRL/zond/transactions"
	"github.com/theQRL/zond/transactions/pool"
	"math/big"
	"path"
	"reflect"
	"sync"
)

type Chain struct {
	lock sync.Mutex

	config *config.Config

	state *state.State
	db2   state2.Database
	//state2 *state2.StateDB

	txPool *pool.TransactionPool

	lastBlock *block.Block
}

func (c *Chain) AccountDB() (*state2.StateDB, error) {
	bm, err := metadata.GetBlockMetaData(c.state.DB(), c.lastBlock.Hash())
	if err != nil {
		log.Error("Failed to load last block metadata")
		return nil, err
	}

	s2, err := state2.New(bm.TrieRoot(), c.db2, nil)
	if err != nil {
		log.Error("Failed to create state2")
		return nil, err
	}
	return s2, nil
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
	if mainChainMetaData.LastBlockHeaderHash().IsEmpty() {
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

func (c *Chain) GetFinalizedHeaderHash() (common.Hash, error) {
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		return common.Hash{}, err
	}
	return mainChainMetaData.FinalizedBlockHeaderHash(), nil
}

func (c *Chain) GetStartingNonFinalizedEpoch() (uint64, error) {
	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		return 0, err
	}
	// Special condition for epoch 0, to start syncing from epoch 0
	if mainChainMetaData.FinalizedBlockSlotNumber() < config.GetDevConfig().BlocksPerEpoch-1 {
		return 0, nil
	}
	finalizedEpoch := mainChainMetaData.FinalizedBlockSlotNumber() / config.GetDevConfig().BlocksPerEpoch

	return finalizedEpoch + 1, nil

}

func (c *Chain) Height() uint64 {
	return c.lastBlock.SlotNumber()
}

func (c *Chain) Load() error {
	db2, err := rawdb.NewLevelDBDatabaseWithFreezer(
		path.Join(c.config.User.DataDir(), c.config.Dev.DB2Name), 16,
		16, path.Join(c.config.User.DataDir(), c.config.Dev.DB2FreezerName),
		c.config.Dev.DB2Name, false)

	if err != nil {
		log.Error("Failed to create db2")
		return err
	}

	c.db2 = state2.NewDatabaseWithConfig(db2, nil)

	db := c.state.DB()
	mainChainMetaData, err := metadata.GetMainChainMetaData(db)
	if err != nil {
		statedb, err := state2.New(common.Hash{}, c.db2, nil)
		if err != nil {
			log.Error("Failed to create statdb")
			return err
		}

		b, err := genesis.GenesisBlock()
		if err != nil {
			log.Error("failed to get genesis block")
			return err
		}

		//blockProposerDilithiumAddress := config.GetDevConfig().Genesis.FoundationDilithiumAddress
		blockHeader := b.Header()
		blockHeaderHash := b.Hash()

		blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()

		epochMetaData := metadata.NewEpochMetaData(0, b.ParentHash(), make([][]byte, 0))

		stateContext, err := state.NewStateContext(db, blockHeader.Number().Uint64(), blockProposerDilithiumPK,
			blockHeader.ParentHash(), blockHeader.ParentHash(), blockHeaderHash,
			b.PartialBlockSigningHash(), b.BlockSigningHash(), epochMetaData)

		if err != nil {
			return err
		}

		stateProcessor := core.NewStateProcessor(&params.ChainConfig{ChainID: c.config.Dev.ChainID}, c.GetBlockHashBySlotNumber)

		preState, err := genesis.LoadPreState()
		if err != nil {
			log.Error("failed to load PreState file")
			return err
		}
		if err := stateProcessor.ProcessGenesisPreState(preState, b, db, statedb); err != nil {
			log.Error("failed to process pre-state")
			return err
		}

		_, _, _, err = stateProcessor.ProcessGenesis(b, statedb, stateContext, vm.Config{})
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

	mainChainLastBlockHash := mainChainMetaData.LastBlockHeaderHash()
	lastBlock, err := block.GetBlock(db, mainChainLastBlockHash)
	if err != nil {
		log.Error("Failed to load last block for ",
			hex.EncodeToString(mainChainLastBlockHash[:]))
		return err
	}
	if lastBlock == nil {
		return errors.New("LastBlock cannot be nil")
	}

	c.lastBlock = lastBlock
	lastBlockHash := c.lastBlock.Hash()
	log.Info(fmt.Sprintf("Current Block Slot Number %d Hash %s",
		c.lastBlock.SlotNumber(), hex.EncodeToString(lastBlockHash[:])))

	return nil
}

func (c *Chain) GetSlotLeaderDilithiumPKBySlotNumber(slotNumber uint64,
	parentHeaderHash common.Hash, parentSlotNumber uint64) ([]byte, error) {
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(),
		slotNumber, parentHeaderHash, parentSlotNumber)
	if err != nil {
		return nil, err
	}

	slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].SlotLeader
	return epochMetaData.Validators()[slotLeaderIndex], nil
}

func (c *Chain) GetAttestorsBySlotNumber(slotNumber uint64,
	parentHeaderHash common.Hash, parentSlotNumber uint64) ([][]byte, error) {
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(), slotNumber,
		parentHeaderHash, parentSlotNumber)
	if err != nil {
		return nil, err
	}

	attestorsIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].Attestors
	validators := epochMetaData.Validators()

	var attestors [][]byte
	for _, attestorIndex := range attestorsIndex {
		attestors = append(attestors, validators[attestorIndex])
	}

	return attestors, nil
}

// GetValidatorsBySlotNumber returns a map of all the validators for a specific slot number.
// The value of map is 1 for slot leader and 0 for the attestors.
func (c *Chain) GetValidatorsBySlotNumber(slotNumber uint64,
	parentHeaderHash common.Hash, parentSlotNumber uint64) (map[string]uint8, error) {
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(),
		slotNumber, parentHeaderHash, parentSlotNumber)
	if err != nil {
		return nil, err
	}

	slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].SlotLeader
	attestorsIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].Attestors
	validators := epochMetaData.Validators()

	validatorsType := make(map[string]uint8)
	slotLeaderPK := epochMetaData.Validators()[slotLeaderIndex]
	validatorsType[hex.EncodeToString(slotLeaderPK[:])] = 1

	for _, attestorIndex := range attestorsIndex {
		validatorPK := validators[attestorIndex]
		validatorsType[hex.EncodeToString(validatorPK[:])] = 0
	}
	return validatorsType, nil
}

func (c *Chain) GetBlockMetaData(headerHash common.Hash) (*metadata.BlockMetaData, error) {
	return metadata.GetBlockMetaData(c.state.DB(), headerHash)
}

func (c *Chain) GetBlock(headerHash common.Hash) (*block.Block, error) {
	return block.GetBlock(c.state.DB(), headerHash)
}

func (c *Chain) GetBlockBySlotNumber(n uint64) (*block.Block, error) {
	panic("not yet implemented")
}

func (c *Chain) GetBlockHashBySlotNumber(n uint64) common.Hash {
	panic("not yet implemented")
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
//			data.HeaderHashes = append(data.HeaderHashes, childBlock.Hash())
//			childHeaderHashes = append(childHeaderHashes, childBlock.Hash())
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
//		data.HeaderHashes = append(data.HeaderHashes, childBlock.Hash())
//		childHeaderHashes = append(childHeaderHashes, childBlock.Hash())
//	}
//
//	for ;len(childHeaderHashes) > 0; {
//		headerHash := childHeaderHashes[0]
//		childHeaderHashes = childHeaderHashes[1:]
//
//		b, err := c.GetBlock(headerHash)
//		if err != nil {
//			log.Error("Failed to GetBlock ", hex.EncodeToString(headerHash))
//			return nil, err
//		}
//
//		blockMetaData, err := c.GetBlockMetaData(headerHash)
//		if err != nil {
//			log.Error("Failed to GetBlockMetaData ", hex.EncodeToString(headerHash))
//			return nil, err
//		}
//
//		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
//			childBlock, err := c.GetBlock(childHeaderHash)
//			if err != nil {
//				log.Error("Error getting child block ", hex.EncodeToString(childHeaderHash))
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
//			data.HeaderHashes = append(data.HeaderHashes, childBlock.Hash())
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

func (c *Chain) ValidateTransaction(protoTx *protos.Transaction) bool {
	statedb, err := c.AccountDB()
	if err != nil {
		log.Error("failed to get statedb, cannot verify transaction")
		return false
	}
	return core.ValidateTransaction(protoTx, statedb)
}

func (c *Chain) ValidateProtocolTransaction(protoTx *protos.ProtocolTransaction, validatorsType map[string]uint8, blockSigningHash common.Hash, isGenesis bool) bool {
	statedb, err := c.AccountDB()
	if err != nil {
		log.Error("failed to get statedb, cannot verify protocol transaction")
		return false
	}
	return core.ValidateProtocolTransaction(protoTx, statedb, validatorsType, blockSigningHash, isGenesis)
}

func (c *Chain) ValidateCoinBaseTransaction(protoTx *protos.ProtocolTransaction, validatorsType map[string]uint8, blockSigningHash common.Hash, isGenesis bool) bool {
	statedb, err := c.AccountDB()
	if err != nil {
		log.Error("failed to get statedb, cannot verify protocol transaction")
		return false
	}
	tx := transactions.CoinBaseTransactionFromPBData(protoTx)
	return core.ValidateCoinBaseTx(tx, statedb, validatorsType, blockSigningHash, isGenesis)
}

func (c *Chain) ValidateAttestTransaction(protoTx *protos.ProtocolTransaction, validatorsType map[string]uint8, partialBlockSigningHash common.Hash) bool {
	statedb, err := c.AccountDB()
	if err != nil {
		log.Error("failed to get statedb, cannot verify protocol transaction")
		return false
	}
	tx := transactions.AttestTransactionFromPBData(protoTx)
	return core.ValidateAttestTx(tx, statedb, validatorsType, partialBlockSigningHash)
}

func (c *Chain) AddBlock(b *block.Block) bool {
	/* TODO: Revise Block Validation */
	maxSlotNumber := c.GetMaxPossibleSlotNumber()
	if b.SlotNumber() > maxSlotNumber {
		log.Error("[AddBlock] Failed to add block as slot number is beyond maximum possible slot number")
		log.Error("MaxPossibleSlotNumber ", maxSlotNumber)
		log.Error("Block Slot Number ", b.SlotNumber())
		return false
	}

	mainChainMetaData, err := metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("[AddBlock] Failed to Get MainChainMetaData ", err.Error())
		return false
	}
	parentBlock, err := c.GetBlock(b.ParentHash())
	if err != nil {
		log.Error("[AddBlock] Failed to Get ParentBlock ", err.Error())
		return false
	}
	if parentBlock.SlotNumber() < mainChainMetaData.FinalizedBlockSlotNumber() {
		log.Error("[AddBlock] ParentBlock slot number is less than finalized block slot number")
		return false
	}

	if parentBlock.SlotNumber() == mainChainMetaData.FinalizedBlockSlotNumber() {
		parentHeaderHash := parentBlock.Hash()
		fHash := mainChainMetaData.FinalizedBlockHeaderHash()
		if !reflect.DeepEqual(parentHeaderHash, mainChainMetaData.FinalizedBlockHeaderHash()) {
			log.Error("[AddBlock] ParentBlock is not the part of the finalized chain",
				" Expected hash ", hex.EncodeToString(fHash[:]),
				" Found hash ", hex.EncodeToString(parentHeaderHash[:]))
			return false
		}
	}

	parentBlockMetaData, err := metadata.GetBlockMetaData(c.state.DB(), b.ParentHash())
	if err != nil {
		log.Error("[AddBlock] Failed to get Parent Block MetaData")
		return false
	}
	epochMetaData, err := block.CalculateEpochMetaData(c.state.DB(), b.Number(),
		b.ParentHash(), parentBlockMetaData.SlotNumber())
	if err != nil {
		log.Error("[AddBlock] Failed to Calculate Epoch MetaData")
		return false
	}

	bHash := b.Hash()
	validators, err := c.GetValidatorsBySlotNumber(b.SlotNumber(), b.ParentHash(), parentBlockMetaData.SlotNumber())
	if err != nil {
		log.Error(fmt.Sprintf("failed to get validatorsBySlotNumber block #%d %s | Error %s", b.SlotNumber(),
			hex.EncodeToString(bHash[:]), err.Error()))
		return false
	}

	blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()
	stateContext, err := state.NewStateContext(c.state.DB(), b.Number(), blockProposerDilithiumPK,
		mainChainMetaData.FinalizedBlockHeaderHash(), b.ParentHash(), b.Hash(),
		b.PartialBlockSigningHash(), b.BlockSigningHash(), epochMetaData)

	statedb, err := state2.New(parentBlockMetaData.TrieRoot(), c.db2, nil)

	// TODO: chain id is currently hardcoded to 0, need to be loaded based on Network type
	stateProcessor := core.NewStateProcessor(&params.ChainConfig{ChainID: c.config.Dev.ChainID}, c.GetBlockHashBySlotNumber)

	//receipts, logs, usedGas, err := stateProcessor.Process(b, statedb, stateContext, validators, false, vm.Config{})
	_, _, _, err = stateProcessor.Process(b, statedb, stateContext, validators, false, vm.Config{})
	if err != nil {
		log.Error(fmt.Sprintf("Failed to process block #%d %s | Error %s", b.SlotNumber(),
			hex.EncodeToString(bHash[:]), err.Error()))
		return false
	}

	//err = b.Commit(c.state.DB(), c.state2, mainChainMetaData.FinalizedBlockHeaderHash(), false)
	//if err != nil {
	//	log.Error(fmt.Sprintf("Failed to commit block #%d %s | Error %s", b.SlotNumber(),
	//		hex.EncodeToString(bHash[:]), err.Error()))
	//	return false
	//}

	// Reload MainChainMetaData from db
	mainChainMetaData, err = metadata.GetMainChainMetaData(c.state.DB())
	if err != nil {
		log.Error("Failed to Get MainChainMetaData ", err.Error())
		return true
	}

	if reflect.DeepEqual(mainChainMetaData.LastBlockHeaderHash(), b.Hash()) {
		c.lastBlock = b
	} else if !reflect.DeepEqual(mainChainMetaData.LastBlockHeaderHash(),
		c.lastBlock.Hash()) {
		lastBlock, err := c.GetBlock(mainChainMetaData.LastBlockHeaderHash())
		lHash := mainChainMetaData.LastBlockHeaderHash()
		if err != nil {
			log.Error("Failed to Get Block for LastBlockHeaderHash ",
				hex.EncodeToString(lHash[:]))
			return true
		}
		c.lastBlock = lastBlock
	}

	log.Info(fmt.Sprintf("Added Block #%d %s", b.SlotNumber(), hex.EncodeToString(bHash[:])))
	return true
}

func (c *Chain) GetStateContext() (*state.StateContext, error) {
	lastBlock := c.lastBlock

	epochMetaData, err := metadata.GetEpochMetaData(c.state.DB(),
		lastBlock.SlotNumber(), lastBlock.ParentHash())
	if err != nil {
		return nil, err
	}

	finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		return nil, err
	}

	return state.
		NewStateContext(c.state.DB(), lastBlock.SlotNumber(), nil,
			finalizedHeaderHash, lastBlock.ParentHash(), c.lastBlock.Hash(),
			lastBlock.PartialBlockSigningHash(), lastBlock.BlockSigningHash(),
			epochMetaData)
}

func (c *Chain) GetStateContext2(slotNumber uint64, blockProposer []byte,
	parentHeaderHash common.Hash, partialBlockSigningHash common.Hash) (*state.StateContext, error) {
	lastBlock := c.lastBlock

	epochMetaData, err := metadata.GetEpochMetaData(c.state.DB(),
		lastBlock.SlotNumber(), lastBlock.ParentHash())
	if err != nil {
		return nil, err
	}

	finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		return nil, err
	}

	return state.NewStateContext(c.state.DB(), slotNumber, blockProposer,
		finalizedHeaderHash, parentHeaderHash, common.Hash{},
		partialBlockSigningHash, common.Hash{},
		epochMetaData)
}

func NewChain(s *state.State) *Chain {
	return &Chain{
		config: config.GetConfig(),
		state:  s,
		db2:    nil,
		txPool: pool.CreateTransactionPool(),
	}
}
