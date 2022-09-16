package chain

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"path"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/block/genesis"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/core"
	"github.com/theQRL/zond/core/rawdb"
	state2 "github.com/theQRL/zond/core/state"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/core/vm"
	"github.com/theQRL/zond/core/vm/runtime"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/params"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/rlp"
	"github.com/theQRL/zond/state"
	"github.com/theQRL/zond/transactions"
	"github.com/theQRL/zond/transactions/pool"
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

// setDefaults to be removed after figuring out better way to call
// runtime vmenv
func setDefaults(cfg *runtime.Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      new(big.Int),
			DAOForkBlock:        new(big.Int),
			DAOForkSupport:      false,
			EIP150Block:         new(big.Int),
			EIP150Hash:          common.Hash{},
			EIP155Block:         new(big.Int),
			EIP158Block:         new(big.Int),
			ByzantiumBlock:      new(big.Int),
			ConstantinopleBlock: new(big.Int),
			PetersburgBlock:     new(big.Int),
			IstanbulBlock:       new(big.Int),
			MuirGlacierBlock:    new(big.Int),
			BerlinBlock:         new(big.Int),
			LondonBlock:         new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
	if cfg.BaseFee == nil {
		cfg.BaseFee = big.NewInt(params.InitialBaseFee)
	}
}

func (c *Chain) Config() *params.ChainConfig {
	cfg := new(runtime.Config)
	setDefaults(cfg)
	return cfg.ChainConfig
}

// GetVMConfig returns the chain VM config.
func (c *Chain) GetVMConfig() *vm.Config {
	return &vm.Config{
		EnablePreimageRecording: false,
	}
}

func (c *Chain) CurrentBlock() *block.Block {
	return c.lastBlock
}

func (c *Chain) CurrentFinalizedBlock() *block.Block {
	blockHash, err := c.GetFinalizedHeaderHash()
	if err != nil {
		log.Error("failed to get finalized header hash")
		return nil
	}

	b, err := c.GetBlock(blockHash)
	if err != nil {
		log.Error("failed to get block for blockHash ", blockHash)
		return nil
	}

	return b
}

func (c *Chain) GetBlockByNumber(number uint64) *block.Block {
	b, err := block.GetBlockByNumber(c.state.DB(), number)
	if err != nil {
		log.Error("failed to get block for slot number ", number)
		return nil
	}

	return b
}

func (c *Chain) GetTransactionMetaDataByHash(txHash common.Hash) (*protos.Transaction, common.Hash, uint64, uint64) {
	txMetaData, err := metadata.GetTransactionMetaData(c.state.DB(), txHash)
	if err != nil {
		log.Error("failed to get transaction metadata for txHash ", txHash)
		return nil, common.Hash{}, 0, 0
	}

	return txMetaData.Transaction(), txMetaData.BlockHash(), txMetaData.BlockNumber(), txMetaData.Index()
}

func (c *Chain) GetReceiptsByHash(headerHash common.Hash, isProtocolTransaction bool) types.Receipts {
	b, err := block.GetBlock(c.state.DB(), headerHash)
	if err != nil {
		log.Error("Failed to derive block", "hash", headerHash, "err", err)
		return nil
	}
	blockReceipts, err := metadata.GetBlockReceipts(c.state.DB(), headerHash, b.SlotNumber(), isProtocolTransaction)
	if err != nil {
		log.Error("Failed to get block receipts", "hash", headerHash, "number", b.SlotNumber(), "err", err)
		return nil
	}

	data := blockReceipts.Receipts()
	if len(data) == 0 {
		return nil
	}
	storageReceipts := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &storageReceipts); err != nil {
		log.Error("Invalid receipt array RLP", "hash", headerHash, "err", err)
		return nil
	}
	receipts := make(types.Receipts, len(storageReceipts))
	for i, storageReceipt := range storageReceipts {
		receipts[i] = (*types.Receipt)(storageReceipt)
	}

	if !isProtocolTransaction {
		if err := receipts.DeriveFieldsForTransactions(headerHash, b.SlotNumber(), b.Transactions()); err != nil {
			log.Error("Failed to derive transaction block receipts fields", "hash", headerHash, "number", b.SlotNumber(), "err", err)
			return nil
		}
	} else {
		if err := receipts.DeriveFieldsForProtocolTransactions(headerHash, b.SlotNumber(), b.ProtocolTransactions()); err != nil {
			log.Error("Failed to derive protocol transaction block receipts fields", "hash", headerHash, "number", b.SlotNumber(), "err", err)
			return nil
		}
	}

	return receipts
}

func (c *Chain) GetLogsByHash(headerHash common.Hash) ([][]*types.Log, error) {
	b, err := block.GetBlock(c.state.DB(), headerHash)
	if err != nil {
		log.Error("Failed to derive block", "hash", headerHash, "err", err)
		return nil, err
	}
	blockReceipts, err := metadata.GetBlockReceipts(c.state.DB(), headerHash, b.SlotNumber(), false)
	if err != nil {
		log.Error("Failed to get block receipts", "hash", headerHash, "number", b.SlotNumber(), "err", err)
		return nil, err
	}

	data := blockReceipts.Receipts()
	return rawdb.ReadLogsFromReceiptsRLPData(data, b.Hash(), b.Number(), b.Transactions()), nil
}

func (c *Chain) StateAt(root common.Hash) (*state2.StateDB, error) {
	//return state2.New(root, c.db2, bc.snaps)
	return state2.New(root, c.db2, nil)
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

func (c *Chain) AccountDBForTrie(trieRoot common.Hash) (*state2.StateDB, error) {
	s2, err := state2.New(trieRoot, c.db2, nil)
	if err != nil {
		log.Error("Failed to create state2")
		return nil, err
	}
	return s2, nil
}

func (c *Chain) EVMCall(contractAddress common.Address, data []byte) ([]byte, error) {
	cfg := new(runtime.Config)
	setDefaults(cfg)
	statedb, err := c.AccountDB()
	if err != nil {
		return nil, err
	}

	cfg.State = statedb
	vmenv := runtime.NewEnv(cfg)
	sender := vm.AccountRef(cfg.Origin)

	ret, _, err := vmenv.Call(
		sender,
		contractAddress,
		data,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, err
}

func (c *Chain) GetMaxPossibleSlotNumber() uint64 {
	d := config.GetDevConfig()
	currentTimestamp := ntp.GetNTP().Time()
	genesisTimestamp := d.Genesis.GenesisTimestamp

	return ((currentTimestamp - genesisTimestamp) / d.BlockTime) + 1
}

func (c *Chain) GetLastBlock() *block.Block {
	return c.lastBlock
}

func (c *Chain) GetTransactionPool() *pool.TransactionPool {
	return c.txPool
}

func (c *Chain) GetNonce(address common.Address) (uint64, error) {
	// TODO (cyyber): doesn't consider the pending nonce for the address involved in Staking
	nonce, found := c.txPool.GetNonceByAddress(address)
	if found {
		return nonce, nil
	}
	statedb, err := c.AccountDB()
	if err != nil {
		log.Error("failed to get accountdb")
		return 0, err
	}
	return statedb.GetNonce(address), nil
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

		//blockProposerDilithiumAddress := config.GetDevConfig().Genesis.FoundationDilithiumAddress
		blockHeader := b.Header()
		blockHeaderHash := b.Hash()

		blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()

		epochMetaData := metadata.NewEpochMetaData(0, b.ParentHash(), make([][]byte, 0))
		epochPBData := epochMetaData.PBData()
		epochPBData.SlotInfo = append(epochPBData.SlotInfo, &protos.SlotInfo{SlotLeader: 0})
		epochMetaData.AddValidators(blockProposerDilithiumPK)

		stateContext, err := state.NewStateContext(db, blockHeader.Number().Uint64(), blockProposerDilithiumPK,
			blockHeader.ParentHash(), blockHeader.ParentHash(), blockHeaderHash,
			b.PartialBlockSigningHash(), b.BlockSigningHash(), epochMetaData)

		if err != nil {
			return err
		}

		// Inject block proposer of genesis block
		stateContext.PrepareValidators(blockProposerDilithiumPK)

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
			misc.BytesToHexStr(mainChainLastBlockHash[:]))
		return err
	}
	if lastBlock == nil {
		return errors.New("LastBlock cannot be nil")
	}

	c.lastBlock = lastBlock
	lastBlockHash := c.lastBlock.Hash()
	log.Info(fmt.Sprintf("Current Block Slot Number %d Hash %s",
		c.lastBlock.SlotNumber(), misc.BytesToHexStr(lastBlockHash[:])))

	return nil
}

func (c *Chain) GetEpochMetaData(trieRoot common.Hash,
	slotNumber uint64, parentHeaderHash common.Hash) (*metadata.EpochMetaData, error) {
	statedb, err := c.AccountDBForTrie(trieRoot)
	if err != nil {
		return nil, err
	}

	return c.CalculateEpochMetaData(statedb, slotNumber, parentHeaderHash)
}

func (c *Chain) GetSlotLeaderDilithiumPKBySlotNumber(trieRoot common.Hash,
	slotNumber uint64, parentHeaderHash common.Hash) ([]byte, error) {

	epochMetaData, err := c.GetEpochMetaData(trieRoot, slotNumber, parentHeaderHash)
	if err != nil {
		return nil, err
	}

	slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].SlotLeader
	return epochMetaData.Validators()[slotLeaderIndex], nil
}

func (c *Chain) GetAttestorsBySlotNumber(trieRoot common.Hash,
	slotNumber uint64, parentHeaderHash common.Hash) ([][]byte, error) {
	epochMetaData, err := c.GetEpochMetaData(trieRoot, slotNumber, parentHeaderHash)
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

func (c *Chain) GetValidators() (*metadata.EpochMetaData, error) {
	currentBlock := c.CurrentBlock()
	return metadata.GetEpochMetaData(c.state.DB(), currentBlock.SlotNumber(), currentBlock.ParentHash())
}

// GetSlotValidatorsMetaDataBySlotNumber returns a map of all the validators for a specific slot number.
// The value of map is 1 for slot leader and 0 for the attestors.
func (c *Chain) GetSlotValidatorsMetaDataBySlotNumber(trieRoot common.Hash,
	slotNumber uint64, parentHeaderHash common.Hash) (*metadata.SlotValidatorsMetaData, error) {

	statedb, err := c.AccountDBForTrie(trieRoot)
	if err != nil {
		return nil, err
	}

	epochMetaData, err := c.CalculateEpochMetaData(statedb, slotNumber, parentHeaderHash)
	if err != nil {
		return nil, err
	}

	slotValidatorsMetaData := metadata.NewSlotValidatorsMetaData(slotNumber, epochMetaData)
	//slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].SlotLeader
	//attestorsIndex := epochMetaData.SlotInfo()[slotNumber%c.config.Dev.BlocksPerEpoch].Attestors
	//validators := epochMetaData.Validators()
	//
	//validatorsType := make(map[string]uint8)
	//slotLeaderPK := epochMetaData.Validators()[slotLeaderIndex]
	//validatorsType[misc.BytesToHexStr(slotLeaderPK[:])] = 1
	//
	//for _, attestorIndex := range attestorsIndex {
	//	validatorPK := validators[attestorIndex]
	//	validatorsType[misc.BytesToHexStr(validatorPK[:])] = 0
	//}
	return slotValidatorsMetaData, nil
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
	blocks, err := block.GetBlockByNumber(c.state.DB(), n)
	if err != nil {
		return common.Hash{}
	}
	return blocks.Hash()
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
//			log.Error("Failed to GetBlock ", misc.BytesToHexStr(headerHash))
//			return nil, err
//		}
//
//		blockMetaData, err := c.GetBlockMetaData(headerHash)
//		if err != nil {
//			log.Error("Failed to GetBlockMetaData ", misc.BytesToHexStr(headerHash))
//			return nil, err
//		}
//
//		for _, childHeaderHash := range blockMetaData.ChildHeaderHashes() {
//			childBlock, err := c.GetBlock(childHeaderHash)
//			if err != nil {
//				log.Error("Error getting child block ", misc.BytesToHexStr(childHeaderHash))
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

func (c *Chain) ValidateTransaction(protoTx *protos.Transaction) error {
	statedb, err := c.AccountDB()
	if err != nil {
		return fmt.Errorf("failed to get statedb, cannot verify transaction %v", err.Error())
	}
	//dec, err := misc.HexStrToBytes("6EDEA5b4fBAd96789433675c49a120b537413296")
	//if err != nil {
	//	log.Error("error decoding string")
	//
	//}
	//var a common.Address
	//copy(a[:], dec)
	//fmt.Println(" >> code >> ", statedb.GetCodeSize(a))
	return core.ValidateTransaction(protoTx, statedb)
}

func (c *Chain) ValidateProtocolTransaction(protoTx *protos.ProtocolTransaction, slotValidatorsMetaData *metadata.SlotValidatorsMetaData, blockSigningHash common.Hash, slotNumber, parentSlotNumber uint64, isGenesis bool) error {
	statedb, err := c.AccountDB()
	if err != nil {
		return fmt.Errorf("failed to get statedb, cannot verify protocol transaction")
	}
	return core.ValidateProtocolTransaction(protoTx, statedb, slotValidatorsMetaData, blockSigningHash, slotNumber, parentSlotNumber, isGenesis)
}

func (c *Chain) ValidateCoinBaseTransaction(protoTx *protos.ProtocolTransaction, validatorsType *metadata.SlotValidatorsMetaData, blockSigningHash common.Hash, slotNumber, parentSlotNumber uint64, isGenesis bool) error {
	statedb, err := c.AccountDB()
	if err != nil {
		return fmt.Errorf("failed to get statedb, cannot verify protocol transaction")
	}
	tx := transactions.CoinBaseTransactionFromPBData(protoTx)
	return core.ValidateCoinBaseTx(tx, statedb, validatorsType, blockSigningHash, slotNumber, parentSlotNumber, isGenesis)
}

func (c *Chain) ValidateAttestTransaction(protoTx *protos.ProtocolTransaction, validatorsType *metadata.SlotValidatorsMetaData, partialBlockSigningHash common.Hash, slotNumber, parentSlotNumber uint64) error {
	statedb, err := c.AccountDB()
	if err != nil {
		return fmt.Errorf("failed to get statedb, cannot verify protocol transaction")
	}
	tx := transactions.AttestTransactionFromPBData(protoTx)
	return core.ValidateAttestTx(tx, statedb, validatorsType, partialBlockSigningHash, slotNumber, parentSlotNumber)
}

func (c *Chain) AddBlock(b *block.Block) bool {
	/* TODO: Revise Block Validation */
	expectedHash := block.ComputeBlockHash(b)
	if b.Hash() != expectedHash {
		log.Error("[AddBlock] Invalid block hash")
		log.Error("Expected block hash ", expectedHash)
		log.Error("Found block hash ", b.Hash())
		return false
	}

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
				" Expected hash ", misc.BytesToHexStr(fHash[:]),
				" Found hash ", misc.BytesToHexStr(parentHeaderHash[:]))
			return false
		}
	}

	parentBlockMetaData, err := metadata.GetBlockMetaData(c.state.DB(), b.ParentHash())
	if err != nil {
		log.Error("[AddBlock] Failed to get Parent Block MetaData")
		return false
	}

	trieRoot := parentBlockMetaData.TrieRoot()
	statedb, err := state2.New(parentBlockMetaData.TrieRoot(), c.db2, nil)

	epochMetaData, err := c.CalculateEpochMetaData(statedb, b.Number(), b.ParentHash())
	if err != nil {
		log.Error("[AddBlock] Failed to Calculate Epoch MetaData")
		return false
	}

	bHash := b.Hash()
	validators, err := c.GetSlotValidatorsMetaDataBySlotNumber(trieRoot, b.SlotNumber(), b.ParentHash())
	if err != nil {
		log.Error(fmt.Sprintf("failed to get validatorsBySlotNumber block #%d %s | Error %s", b.SlotNumber(),
			misc.BytesToHexStr(bHash[:]), err.Error()))
		return false
	}

	blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()
	stateContext, err := state.NewStateContext(c.state.DB(), b.Number(), blockProposerDilithiumPK,
		mainChainMetaData.FinalizedBlockHeaderHash(), b.ParentHash(), b.Hash(),
		b.PartialBlockSigningHash(), b.BlockSigningHash(), epochMetaData)

	// TODO: chain id is currently hardcoded to 0, need to be loaded based on Network type
	stateProcessor := core.NewStateProcessor(&params.ChainConfig{ChainID: c.config.Dev.ChainID}, c.GetBlockHashBySlotNumber)

	//receipts, logs, usedGas, err := stateProcessor.Process(b, statedb, stateContext, validators, false, vm.Config{})
	_, _, _, err = stateProcessor.Process(b, parentBlock, statedb, stateContext, validators, false, vm.Config{})
	if err != nil {
		log.Error(fmt.Sprintf("Failed to process block #%d %s | Error %s", b.SlotNumber(),
			misc.BytesToHexStr(bHash[:]), err.Error()))
		return false
	}

	//err = b.Commit(c.state.DB(), c.state2, mainChainMetaData.FinalizedBlockHeaderHash(), false)
	//if err != nil {
	//	log.Error(fmt.Sprintf("Failed to commit block #%d %s | Error %s", b.SlotNumber(),
	//		misc.BytesToHexStr(bHash[:]), err.Error()))
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
				misc.BytesToHexStr(lHash[:]))
			return true
		}
		c.lastBlock = lastBlock
	}

	if b.Hash() == c.lastBlock.Hash() && b.Epoch() != parentBlock.Epoch() {
		err = b.UpdateFinalizedEpoch(c.state.DB(), stateContext)
		if err != nil {
			log.Error("Failed to update finalized epoch")
		} else {
			finalizedHeaderHash, err := c.GetFinalizedHeaderHash()
			if err != nil {
				log.Error("Error getting finalized header hash")
			} else {
				b, err := c.GetBlock(finalizedHeaderHash)
				if err != nil {
					log.Error("Error getting block for finalized header hash")
				} else {
					log.Info(fmt.Sprintf("Updated finalized epoch to Epoch %d Slot Number #%d %s ", b.Epoch(), b.SlotNumber(), b.Hash()))
				}
			}
		}
	}

	log.Info(fmt.Sprintf("Added Block #%d %s", b.SlotNumber(), misc.BytesToHexStr(bHash[:])))
	log.Info(fmt.Sprintf("Protocol Txs Count %d | Txs Count %d",
		len(b.ProtocolTransactions()), len(b.Transactions())))
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

func (c *Chain) CalculateEpochMetaData(statedb *state2.StateDB, slotNumber uint64,
	parentHeaderHash common.Hash) (*metadata.EpochMetaData, error) {

	db := c.state.DB()
	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	parentBlockMetaData, err := metadata.GetBlockMetaData(db, parentHeaderHash)
	if err != nil {
		return nil, err
	}
	parentEpoch := parentBlockMetaData.SlotNumber() / blocksPerEpoch
	epoch := slotNumber / blocksPerEpoch

	if parentEpoch == epoch {
		return metadata.GetEpochMetaData(db, slotNumber, parentHeaderHash)
	}

	epoch = parentEpoch
	var pathToFirstBlockOfEpoch []common.Hash
	if parentBlockMetaData.SlotNumber() == 0 {
		pathToFirstBlockOfEpoch = append(pathToFirstBlockOfEpoch, parentBlockMetaData.HeaderHash())
	} else {
		for epoch == parentEpoch {
			pathToFirstBlockOfEpoch = append(pathToFirstBlockOfEpoch, parentBlockMetaData.HeaderHash())
			if parentBlockMetaData.SlotNumber() == 0 {
				break
			}
			parentBlockMetaData, err = metadata.GetBlockMetaData(db, parentBlockMetaData.ParentHeaderHash())
			if err != nil {
				return nil, err
			}
			parentEpoch = parentBlockMetaData.SlotNumber() / blocksPerEpoch
		}
	}

	lenPathToFirstBlockOfEpoch := len(pathToFirstBlockOfEpoch)
	if lenPathToFirstBlockOfEpoch == 0 {
		return nil, errors.New("lenPathToFirstBlockOfEpoch is 0")
	}

	firstBlockOfEpochHeaderHash := pathToFirstBlockOfEpoch[lenPathToFirstBlockOfEpoch-1]
	blockMetaData, err := metadata.GetBlockMetaData(db, firstBlockOfEpochHeaderHash)
	if err != nil {
		return nil, err
	}

	epochMetaData, err := metadata.GetEpochMetaData(db, blockMetaData.SlotNumber(),
		blockMetaData.ParentHeaderHash())

	if err != nil {
		return nil, err
	}

	totalStakeAmountAllotted := uint64(len(epochMetaData.Validators())) * config.GetDevConfig().StakeAmount
	epochMetaData.UpdatePrevEpochStakeData(0,
		totalStakeAmountAllotted)

	var pendingStakeValidatorsUpdate [][]byte
	for i := lenPathToFirstBlockOfEpoch - 1; i >= 0; i-- {
		b, err := block.GetBlock(db, pathToFirstBlockOfEpoch[i])
		if err != nil {
			return nil, err
		}
		err = core.ProcessEpochMetaData(b, statedb, epochMetaData)
		if err != nil {
			return nil, err
		}
		// Ignore genesis block otherwise it will result into issue as we have
		// already processed while committing genesis block
		if b.SlotNumber() != 0 {
			pendingStakeValidatorsUpdate = append(pendingStakeValidatorsUpdate, b.GetPendingValidatorsUpdate()...)
		}
	}

	err = core.UpdateStakeValidators(statedb, pendingStakeValidatorsUpdate, epochMetaData)
	if err != nil {
		return nil, err
	}

	// TODO: load accountState of all the dilithium pk in validatorsStateChanged
	// then update the stake balance, pending stake balance and the balance
	// accountState must be loaded based on the trie of parentHeaderHash

	randomSeed := core.GetRandomSeed(parentHeaderHash)

	currentEpoch := slotNumber / blocksPerEpoch
	epochMetaData.AllotSlots(randomSeed, currentEpoch, parentHeaderHash)

	return epochMetaData, nil
}
