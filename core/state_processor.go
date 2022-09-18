package core

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/block/rewards"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/common/math"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/core/state"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/core/vm"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/errmsg"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/params"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/rlp"
	state2 "github.com/theQRL/zond/state"
	"github.com/theQRL/zond/storagekeys"
	"github.com/theQRL/zond/transactions"
	"go.etcd.io/bbolt"
	"math/big"
	"reflect"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config      *params.ChainConfig // Chain configuration options
	getHashFunc vm.GetHashFunc
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, getHashFunc vm.GetHashFunc) *StateProcessor {
	return &StateProcessor{
		config:      config,
		getHashFunc: getHashFunc,
	}
}

func (p *StateProcessor) ProcessGenesisPreState(preState *protos.PreState, b *block.Block, db *db.DB, statedb *state.StateDB) error {
	m := metadata.NewMainChainMetaData(common.Hash{}, 0,
		common.Hash{}, 0)
	totalStakeAmount, _ := big.NewInt(0).MarshalText()
	bm := metadata.NewBlockMetaData(common.Hash{},
		b.ParentHash(), 0, totalStakeAmount, common.Hash{})
	err := db.DB().Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		if err := m.Commit(b); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(bm.HeaderHash())); err != nil {
			return err
		}
		return bm.Commit(b)
	})
	if err != nil {
		return err
	}

	blockProposerDilithiumAddress := misc.GetDilithiumAddressFromUnSizedPK(b.ProtocolTransactions()[0].GetPk())
	statedb.GetOrNewStateObject(blockProposerDilithiumAddress).SetBalance(big.NewInt(int64(config.GetDevConfig().Genesis.SuppliedCoins)))

	if blockProposerDilithiumAddress != config.GetDevConfig().Genesis.FoundationDilithiumAddress {
		expectedFoundationDilithiumAddress := misc.BytesToHexStr(config.GetDevConfig().Genesis.FoundationDilithiumAddress[:])

		log.Warn("block proposer dilithium address is not matching with the foundation dilithium address in config")
		log.Warn("expected foundation dilithium address ", expectedFoundationDilithiumAddress)
		log.Warn("found foundation dilithium address ", blockProposerDilithiumAddress)

		return fmt.Errorf("failed to process genesis pre state")
	}

	for _, addressBalance := range preState.AddressBalance {
		var address common.Address
		copy(address[:], addressBalance.Address)
		statedb.GetOrNewStateObject(address).SetBalance(big.NewInt(int64(addressBalance.Balance)))
	}

	return nil
}

func (p *StateProcessor) ProcessGenesis(b *block.Block, statedb *state.StateDB, stateContext *state2.StateContext, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	//validatorsType := make(map[string]uint8)

	//blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()
	//validatorsType[misc.BytesToHexStr(blockProposerDilithiumPK)] = 1
	slotValidatorsMetaData := metadata.NewSlotValidatorsMetaData(b.SlotNumber(), stateContext.GetEpochMetaData())

	var (
		protocolTxReceipts types.Receipts
		txReceipts         types.Receipts
		usedGas            = new(uint64)
		header             = b.Header()
		blockHash          = b.Hash()
		blockNumber        = big.NewInt(int64(b.Number()))
		allLogs            []*types.Log
		gp                 = new(GasPool).AddGas(b.GasLimit())
	)

	blockContext := NewEVMBlockContext(header, p.getHashFunc, b.Minter())
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)

	switch b.ProtocolTransactions()[0].Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
	default:
		panic("coinbase transaction not found at index 0 of the protocol transactions in the b")
	}

	// Code to process protocol transactions
	for i, protoTx := range b.ProtocolTransactions() {
		//tx := transactions.ProtoToProtocolTransaction(protoTx)
		var receipt *types.Receipt
		var err error

		switch protoTx.Type.(type) {
		case *protos.ProtocolTransaction_CoinBase:
			if i != 0 {
				panic("multiple coinbase transaction found")
			}

			coinBaseTx := transactions.CoinBaseTransactionFromPBData(b.ProtocolTransactions()[0])

			if err := ValidateCoinBaseTx(coinBaseTx, statedb, slotValidatorsMetaData, b.BlockSigningHash(), b.SlotNumber(), b.SlotNumber(), true); err != nil {
				return nil, nil, 0, err
			}

			coinBaseTxHash := coinBaseTx.TxHash(coinBaseTx.GetSigningHash(b.BlockSigningHash()))
			statedb.Prepare(coinBaseTxHash, 0)
			receipt, err = applyCoinBaseTransaction(statedb, stateContext, blockNumber, blockHash, coinBaseTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply coinbase tx %d [%v]: %w", 0, coinBaseTxHash, err)
			}

		case *protos.ProtocolTransaction_Attest:
			attestTx := transactions.AttestTransactionFromPBData(protoTx)

			if err := ValidateAttestTx(attestTx, statedb, slotValidatorsMetaData, b.PartialBlockSigningHash(), b.SlotNumber(), b.SlotNumber()); err != nil {
				return nil, nil, 0, err
			}

			txHash := attestTx.TxHash(attestTx.GetSigningHash(b.PartialBlockSigningHash()))
			statedb.Prepare(txHash, i)
			receipt, err = applyAttestTransaction(statedb, stateContext, blockNumber, blockHash, attestTx,
				b.ProtocolTransactions()[0].GetCoinBase().AttestorReward)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply attest tx %d [%v]: %w", 0, txHash, err)
			}
		}
		protocolTxReceipts = append(protocolTxReceipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	// Iterate over and process the individual transactions
	for i, protoTx := range b.Transactions() {
		tx := transactions.ProtoToTransaction(protoTx)
		statedb.Prepare(tx.Hash(), i)

		var receipt *types.Receipt
		var err error

		switch protoTx.Type.(type) {
		case *protos.Transaction_Stake:
			stakeTx := transactions.StakeTransactionFromPBData(protoTx)

			if err := ValidateStakeTxn(stakeTx, statedb); err != nil {
				return nil, nil, 0, err
			}
			receipt, err = applyStakeTransaction(gp, statedb, blockNumber, blockHash, stakeTx, usedGas, b.Minter())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply stake tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}

		case *protos.Transaction_Transfer:
			transferTx := transactions.TransferTransactionFromPBData(protoTx)
			if err := ValidateTransferTxn(transferTx, statedb); err != nil {
				return nil, nil, 0, err
			}
			msg, err := TransferTxAsMessage(transferTx, header.BaseFee())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
			receipt, err = applyTransaction(msg, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply transfer tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
		}
		txReceipts = append(txReceipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)

		// If not a contract and an xmss address then update the ots bitfield
		if statedb.GetCodeSize(tx.AddrFrom()) == 0 && xmss.IsValidXMSSAddress(tx.AddrFrom()) {
			statedb.SetOTSBitfield(tx.AddrFrom(), misc.GetOTSIndexFromSignature(tx.Signature()), false)
		}

	}

	epochMetaData := stateContext.GetEpochMetaData()
	err := ProcessEpochMetaData(b, statedb, epochMetaData)
	if err != nil {
		log.Error("Failed to Process Epoch MetaData")
		return nil, nil, 0, err
	}

	pendingStakeValidatorsUpdate := b.GetPendingValidatorsUpdate()

	err = UpdateStakeValidators(statedb, pendingStakeValidatorsUpdate, epochMetaData)
	if err != nil {
		log.Error("Failed to update stake validators for genesis")
		return nil, nil, 0, err
	}

	// For Genesis Block total stake found and alloted must be same
	epochMetaData.UpdatePrevEpochStakeData(epochMetaData.TotalStakeAmountFound(),
		epochMetaData.TotalStakeAmountFound())

	randomSeed := GetRandomSeed(b.ParentHash())

	currentEpoch := uint64(0)
	epochMetaData.AllotSlots(randomSeed, currentEpoch, b.ParentHash())

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	//p.engine.Finalize(header, statedb)

	statedb.Finalise(true)
	trieRoot, err := statedb.Commit(true)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit stateDB : %w", err)
	}

	err = statedb.Database().TrieDB().Commit(trieRoot, true, nil)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit trieDB : %w", err)
	}

	/* Set block bloom before commitment */
	protocolTxBloom := types.CreateBloom(protocolTxReceipts)
	txBloom := types.CreateBloom(txReceipts)
	b.UpdateBloom(protocolTxBloom, txBloom)
	b.UpdateRoots(trieRoot)

	err = p.Commit(stateContext, b, protocolTxReceipts, txReceipts, trieRoot, true)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit statecontext : %w", err)
	}

	return txReceipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) Process(b, parentBlock *block.Block, statedb *state.StateDB, stateContext *state2.StateContext, slotValidatorsMetaData *metadata.SlotValidatorsMetaData, isFinalizedState bool, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		protocolTxReceipts types.Receipts
		txReceipts         types.Receipts
		usedGas            = new(uint64)
		header             = b.Header()
		blockHash          = b.Hash()
		blockNumber        = big.NewInt(int64(b.Number()))
		allLogs            []*types.Log
		gp                 = new(GasPool).AddGas(b.GasLimit())
	)

	blockContext := NewEVMBlockContext(header, p.getHashFunc, b.Minter())
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)

	switch b.ProtocolTransactions()[0].Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
	default:
		panic("coinbase transaction not found at index 0 of the protocol transactions in the b")
	}

	// Code to process protocol transactions
	for i, protoTx := range b.ProtocolTransactions() {
		//tx := transactions.ProtoToProtocolTransaction(protoTx)
		var receipt *types.Receipt
		var err error

		switch protoTx.Type.(type) {
		case *protos.ProtocolTransaction_CoinBase:
			if i != 0 {
				panic("multiple coinbase transaction found")
			}

			coinBaseTx := transactions.CoinBaseTransactionFromPBData(b.ProtocolTransactions()[0])

			if err := ValidateCoinBaseTx(coinBaseTx, statedb, slotValidatorsMetaData, b.BlockSigningHash(), b.SlotNumber(), parentBlock.SlotNumber(), false); err != nil {
				return nil, nil, 0, err
			}

			coinBaseTxHash := coinBaseTx.TxHash(coinBaseTx.GetSigningHash(b.BlockSigningHash()))
			statedb.Prepare(coinBaseTxHash, i)
			receipt, err = applyCoinBaseTransaction(statedb, stateContext, blockNumber, blockHash, coinBaseTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply coinbase tx %d [%v]: %w", 0, coinBaseTxHash, err)
			}

		case *protos.ProtocolTransaction_Attest:
			attestTx := transactions.AttestTransactionFromPBData(protoTx)

			if err := ValidateAttestTx(attestTx, statedb, slotValidatorsMetaData, b.PartialBlockSigningHash(), b.SlotNumber(), parentBlock.SlotNumber()); err != nil {
				return nil, nil, 0, err
			}

			txHash := attestTx.TxHash(attestTx.GetSigningHash(b.PartialBlockSigningHash()))
			statedb.Prepare(txHash, i)
			receipt, err = applyAttestTransaction(statedb, stateContext, blockNumber, blockHash, attestTx,
				b.ProtocolTransactions()[0].GetCoinBase().AttestorReward)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply attest tx %d [%v]: %w", 0, txHash, err)
			}
		}
		protocolTxReceipts = append(protocolTxReceipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	// Iterate over and process the individual transactions
	for i, protoTx := range b.Transactions() {
		tx := transactions.ProtoToTransaction(protoTx)
		statedb.Prepare(tx.Hash(), i)

		var receipt *types.Receipt
		var err error

		switch protoTx.Type.(type) {
		case *protos.Transaction_Stake:
			stakeTx := transactions.StakeTransactionFromPBData(protoTx)
			if err := ValidateStakeTxn(stakeTx, statedb); err != nil {
				return nil, nil, 0, err
			}
			receipt, err = applyStakeTransaction(gp, statedb, blockNumber, blockHash, stakeTx, usedGas, b.Minter())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply stake tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}

		case *protos.Transaction_Transfer:
			transferTx := transactions.TransferTransactionFromPBData(protoTx)
			if err := ValidateTransferTxn(transferTx, statedb); err != nil {
				return nil, nil, 0, err
			}
			msg, err := TransferTxAsMessage(transferTx, header.BaseFee())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
			receipt, err = applyTransaction(msg, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply transfer tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
		}
		txReceipts = append(txReceipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)

		// If code size is 0 in that case it is an account and not a contract
		if statedb.GetCodeSize(tx.AddrFrom()) == 0 && xmss.IsValidXMSSAddress(tx.AddrFrom()) {
			statedb.SetOTSBitfield(tx.AddrFrom(), misc.GetOTSIndexFromSignature(tx.Signature()), false)
		}

	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	//p.engine.Finalize(header, statedb)

	statedb.Finalise(true)
	trieRoot, err := statedb.Commit(true)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit stateDB : %w", err)
	}

	err = statedb.Database().TrieDB().Commit(trieRoot, true, nil)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit trieDB : %w", err)
	}

	/* Set block bloom before commitment */
	protocolTxBloom := types.CreateBloom(protocolTxReceipts)
	txBloom := types.CreateBloom(txReceipts)
	b.UpdateBloom(protocolTxBloom, txBloom)
	b.UpdateRoots(trieRoot)

	err = p.Commit(stateContext, b, protocolTxReceipts, txReceipts, trieRoot, isFinalizedState)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit statecontext : %w", err)
	}

	return txReceipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) Commit(s *state2.StateContext, b *block.Block, protocolTxReceipts, TxReceipts types.Receipts, trieRoot common.Hash, isFinalizedState bool) error {
	var parentBlockMetaData *metadata.BlockMetaData
	var err error
	totalStakeAmount := big.NewInt(0)
	lastBlockTotalStakeAmount := big.NewInt(0)

	if s.GetSlotNumber() != 0 {
		parentBlockMetaData, err = metadata.GetBlockMetaData(s.GetDB(), s.GetParentBlockHeaderHash())
		if err != nil {
			log.Error("[Commit] Failed to load Parent BlockMetaData")
			return err
		}
		parentBlockMetaData.AddChildHeaderHash(s.GetBlockHeaderHash())

		err = totalStakeAmount.UnmarshalText(parentBlockMetaData.TotalStakeAmount())
		if err != nil {
			log.Error("[Commit] Unable to unmarshal total stake amount of parent block metadata")
			return err
		}

		lastBlockHash := s.GetMainChainMetaData().LastBlockHeaderHash()
		lastBlockMetaData, err := metadata.GetBlockMetaData(s.GetDB(), lastBlockHash)
		if err != nil {
			log.Error("[Commit] Failed to load last block meta data ",
				misc.BytesToHexStr(lastBlockHash[:]))
			return err
		}
		err = lastBlockTotalStakeAmount.UnmarshalText(lastBlockMetaData.TotalStakeAmount())
		if err != nil {
			log.Error("[Commit] Unable to Unmarshal Text for lastblockmetadata total stake amount ",
				misc.BytesToHexStr(lastBlockHash[:]))
			return err
		}
	}

	totalStakeAmount = totalStakeAmount.Add(totalStakeAmount, s.GetCurrentBlockTotalStakeAmount())
	bytesTotalStakeAmount, err := totalStakeAmount.MarshalText()
	if err != nil {
		log.Error("[Commit] Unable to marshal total stake amount")
		return err
	}

	byteProtocolTxReceipts, err := receiptsToByteReceipts(protocolTxReceipts)
	if err != nil {
		log.Error("[Commit] Failed to convert receiptsToByteReceipts for protocolTx")
		return err
	}

	byteTxReceipts, err := receiptsToByteReceipts(TxReceipts)
	if err != nil {
		log.Error("[Commit] Failed to convert receiptsToByteReceipts for tx")
		return err
	}

	blockMetaData := metadata.NewBlockMetaData(s.GetParentBlockHeaderHash(), s.GetBlockHeaderHash(),
		s.GetSlotNumber(), bytesTotalStakeAmount, trieRoot)
	return s.GetDB().DB().Update(func(tx *bbolt.Tx) error {
		var err error
		bucket := tx.Bucket([]byte("DB"))
		if err := blockMetaData.Commit(bucket); err != nil {
			log.Error("[Commit] Failed to commit BlockMetaData")
			return err
		}

		err = s.GetEpochBlockHashes().AddHeaderHashBySlotNumber(s.GetBlockHeaderHash(), s.GetSlotNumber())
		if err != nil {
			log.Error("[Commit] Failed to Add Hash into EpochBlockHashes")
			return err
		}
		if err := s.GetEpochBlockHashes().Commit(bucket); err != nil {
			log.Error("[Commit] Failed to commit EpochBlockHashes")
			return err
		}

		if s.GetSlotNumber() != 0 {
			if err := parentBlockMetaData.Commit(bucket); err != nil {
				log.Error("[Commit] Failed to commit ParentBlockMetaData")
				return err
			}
		}

		if s.GetSlotNumber() == 0 || blockMetaData.Epoch() != parentBlockMetaData.Epoch() {
			if err := s.GetEpochMetaData().Commit(bucket); err != nil {
				log.Error("[Commit] Failed to commit EpochMetaData")
				return err
			}
		}

		bytesBlock, err := b.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize block : %w", err)
		}
		err = bucket.Put(block.GetBlockStorageKey(b.Hash()), bytesBlock)
		if err != nil {
			log.Error("[Commit] Failed to commit block")
			return err
		}

		pbr := metadata.NewBlockReceipts(byteProtocolTxReceipts)
		bytesProtocolTxReceipt, err := pbr.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize block : %w", err)
		}
		err = bucket.Put(metadata.GetBlockReceiptsKey(b.Hash(), b.Number(), true), bytesProtocolTxReceipt)
		if err != nil {
			log.Error("[Commit] Failed to commit protocol tx receipts")
			return err
		}

		br := metadata.NewBlockReceipts(byteTxReceipts)
		bytesTxReceipt, err := br.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize block : %w", err)
		}
		err = bucket.Put(metadata.GetBlockReceiptsKey(b.Hash(), b.Number(), false), bytesTxReceipt)
		if err != nil {
			log.Error("[Commit] Failed to commit tx receipts")
			return err
		}

		if isFinalizedState {
			// Update Main Chain Finalized Block Data
			s.GetMainChainMetaData().UpdateFinalizedBlockData(s.GetBlockHeaderHash(), s.GetSlotNumber())
			s.GetMainChainMetaData().UpdateLastBlockData(s.GetBlockHeaderHash(), s.GetSlotNumber())
			if err := s.GetMainChainMetaData().Commit(bucket); err != nil {
				log.Error("[Commit] Failed to commit MainChainMetaData")
				return err
			}
		}

		if s.GetSlotNumber() == 0 || totalStakeAmount.Cmp(lastBlockTotalStakeAmount) == 1 {
			// Update Main Chain Last Block Data
			s.GetMainChainMetaData().UpdateLastBlockData(s.GetBlockHeaderHash(), s.GetSlotNumber())
			if err := s.GetMainChainMetaData().Commit(bucket); err != nil {
				log.Error("[Commit] Failed to commit MainChainMetaData")
				return err
			}

			err = bucket.Put([]byte("mainchain-head-trie-root"), trieRoot[:])
			if err != nil {
				log.Error("[Commit] Failed to commit state trie root")
				return err
			}
			blockHeaderHash := s.GetBlockHeaderHash()
			err = bucket.Put(storagekeys.GetBlockHashStorageKeyBySlotNumber(s.GetSlotNumber()), blockHeaderHash[:])

			for i, protoProtocolTx := range b.ProtocolTransactions() {
				protocolTXMetaData := metadata.NewProtocolTransactionMetaData(s.GetBlockHeaderHash(),
					s.GetSlotNumber(), uint64(i), protoProtocolTx)
				if err := protocolTXMetaData.Commit(bucket); err != nil {
					log.Error("[Commit] Failed to commit protocol tx MetaData")
					return err
				}
			}

			for i, protoTx := range b.Transactions() {
				txMetaData := metadata.NewTransactionMetaData(s.GetBlockHeaderHash(),
					s.GetSlotNumber(), uint64(i), protoTx)
				if err := txMetaData.Commit(bucket); err != nil {
					log.Error("[Commit] Failed to commit tx MetaData")
					return err
				}
			}
		}

		if !isFinalizedState {
			bucket, err = tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(s.GetBlockHeaderHash()))
			if err != nil {
				log.Error("[Commit] Failed to create bucket")
				return err
			}
		}

		return nil
	})
}

func applyTransaction(msg types.Message, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx transactions.TransactionInterface, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte

	statedb.Finalise(true)

	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: uint8(tx.Type()), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	/*
		TODO: Origin is derived from AddrFrom() which returns master addr in case txn signed by slave
		and when slave address is used for the txn, the nonce value of the signer increases.
		While in the following case nonce of the master address will be used.

		Origin has to be the signer address.
	*/
	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

func applyStakeTransaction(gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *transactions.Stake, usedGas *uint64, minter *common.Address) (*types.Receipt, error) {
	// Update the state with pending changes.
	var root []byte

	statedb.Finalise(true)

	// TODO: remove hardcoded fixed gas for the stake transaction
	StakeTxGas := uint64(1000)
	*usedGas += StakeTxGas

	/* -- Stake Transaction changes starts -- */
	address := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(address)

	accountState.SetNonce(statedb.GetNonce(address) + 1)

	// Calculate and Subtract Fee
	fee := tx.Gas() * tx.GasPrice()
	accountState.SubBalance(big.NewInt(int64(fee)))

	bigAmount := big.NewInt(int64(tx.Amount()))

	// Add all pending stake balance back to balance
	accountState.AddBalance(accountState.PendingStakeBalance())

	// Subtract Stake amount from the current balance
	accountState.SubBalance(bigAmount)

	// Set pending stake balance to the amount, as we already added the older pending balance to the balance
	accountState.SetPendingStakeBalance(bigAmount)

	// refund remaining gas
	accountState.AddBalance(big.NewInt(int64(tx.Gas()-(StakeTxGas)) * int64(tx.GasPrice())))
	// add gas fee to the block proposer address
	statedb.AddBalance(*minter, big.NewInt(int64((StakeTxGas)*tx.GasPrice())))

	if err := gp.SubGas(StakeTxGas); err != nil {
		return nil, err
	}
	/* -- Stake Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: uint8(tx.Type()), PostState: root, CumulativeGasUsed: *usedGas}
	receipt.Status = types.ReceiptStatusSuccessful

	receipt.TxHash = tx.Hash()
	receipt.GasUsed = *usedGas

	// Set the receipt logs and create the bloom filter.
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, nil
}

func applyCoinBaseTransaction(statedb *state.StateDB, stateContext *state2.StateContext, blockNumber *big.Int, blockHash common.Hash, tx *transactions.CoinBase) (*types.Receipt, error) {
	// Update the state with pending changes.
	var root []byte

	statedb.Finalise(true)

	/* -- Coinbase Transaction changes starts -- */
	address := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(address)

	if err := stateContext.ProcessBlockProposerFlag(tx.PK(), accountState.StakeBalance()); err != nil {
		return nil, fmt.Errorf("failed to process block proposer %s | Reason: %w", misc.BytesToHexStr(tx.PK()), err)
	}

	blockReward := big.NewInt(int64(tx.BlockProposerReward()))
	accountState.AddBalance(blockReward)
	accountState.AddBalance(big.NewInt(int64(tx.FeeReward())))
	accountState.SetNonce(statedb.GetNonce(address) + 1)

	coinBaseAddress := config.GetDevConfig().Genesis.CoinBaseAddress
	coinBaseAccountState := statedb.GetOrNewStateObject(coinBaseAddress)
	coinBaseAccountState.SubBalance(blockReward)

	signedMessage := tx.GetSigningHash(stateContext.BlockSigningHash())
	txHash := tx.TxHash(signedMessage)
	/* -- Coinbase Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: uint8(tx.Type()), PostState: root}
	receipt.Status = types.ReceiptStatusSuccessful

	receipt.TxHash = txHash

	// Set the receipt logs and create the bloom filter.
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, nil
}

func applyAttestTransaction(statedb *state.StateDB, stateContext *state2.StateContext, blockNumber *big.Int, blockHash common.Hash, tx *transactions.Attest, attestorReward uint64) (*types.Receipt, error) {
	// Update the state with pending changes.
	var root []byte

	statedb.Finalise(true)

	/* -- Attest Transaction changes starts -- */
	signedMessage := tx.GetSigningHash(stateContext.PartialBlockSigningHash())
	txHash := tx.TxHash(signedMessage)

	address := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(address)

	if err := stateContext.ProcessAttestorsFlag(tx.PK(), accountState.StakeBalance()); err != nil {
		return nil, fmt.Errorf("failed to process attest transaction for attestor  %s | Reason: %w", misc.BytesToHexStr(tx.PK()), err)
	}

	reward := big.NewInt(int64(attestorReward))
	accountState.AddBalance(reward)
	accountState.SetNonce(statedb.GetNonce(address) + 1)

	coinBaseAddress := config.GetDevConfig().Genesis.CoinBaseAddress
	coinBaseAccountState := statedb.GetOrNewStateObject(coinBaseAddress)
	coinBaseAccountState.SubBalance(reward)
	/* -- Attest Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: uint8(tx.Type()), PostState: root}
	receipt.Status = types.ReceiptStatusSuccessful

	receipt.TxHash = txHash

	// Set the receipt logs and create the bloom filter.
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, nil
}

func ValidateTransferTxn(tx *transactions.Transfer, statedb *state.StateDB) error {
	txHash := tx.GenerateTxHash()

	addrFrom := tx.AddrFrom()
	accountState := statedb.GetOrNewStateObject(addrFrom)

	if tx.Hash() != txHash {
		return fmt.Errorf(errmsg.TXInvalidTxHash, "transfer", tx.Hash(), addrFrom, txHash)
	}

	if tx.Nonce() != accountState.Nonce() {
		return fmt.Errorf(errmsg.TXInvalidNonce, "transfer", tx.Hash(), addrFrom, tx.Nonce(), accountState.Nonce())
	}

	if len(tx.PK()) == xmss.ExtendedPKSize {
		if misc.IsUsedOTSIndex(tx.OTSIndex(), accountState.OTSBitfield()) {
			return fmt.Errorf(errmsg.TXInvalidXMSSOTSIndex, "transfer", txHash, addrFrom, tx.OTSIndex())
		}
	}

	balance := accountState.Balance().Uint64()
	requiredBalance := tx.Value() + tx.Gas()*tx.GasPrice()
	if balance < requiredBalance {
		return fmt.Errorf(errmsg.TXInsufficientBalance, "transfer", tx.Hash(), addrFrom, requiredBalance, balance)
	}

	// TODO: Move to some common validation
	if !(xmss.IsValidXMSSAddress(tx.AddrFrom()) || dilithium.IsValidDilithiumAddress(tx.AddrFrom())) {
		return fmt.Errorf(errmsg.TXInvalidAddrFrom, "transfer", tx.Hash(), tx.AddrFrom())
	}

	if tx.To() != nil && !(xmss.IsValidXMSSAddress(*tx.To()) || dilithium.IsValidDilithiumAddress(*tx.To())) {
		// TODO: check if this conditions holds true for cases where contract can create other address
		// If the address is neither a valid XMSS or Dilithium address and if code size is 0
		// then this address not a contract address, thus must be an invalid address
		if statedb.GetCodeSize(*tx.To()) == 0 {
			return fmt.Errorf(errmsg.TXInvalidAddrTo, "transfer", tx.Hash(), tx.To())
		}
	}

	if reflect.DeepEqual(tx.AddrFrom(), config.GetDevConfig().Genesis.CoinBaseAddress) {
		return fmt.Errorf(errmsg.TXAddrFromCannotBeCoinBaseAddr, "transfer", tx.Hash(), tx.AddrFrom())
	}

	if tx.To() != nil && reflect.DeepEqual(*tx.To(), config.GetDevConfig().Genesis.CoinBaseAddress) {
		return fmt.Errorf(errmsg.TXAddrToCannotBeCoinBaseAddr, "transfer", tx.Hash(), tx.To())
	}

	dataLen := len(tx.Data())
	// TODO (cyyber): Move the hardcoded value to config
	dataLenLimit := 24 * 1024
	if dataLen > dataLenLimit {
		return fmt.Errorf(errmsg.TXDataLengthExceedsLimit, "transfer", tx.Hash(), addrFrom, dataLen, dataLenLimit)
	}

	signingHash := tx.GetSigningHash()
	if len(tx.PK()) == dilithium.PKSizePacked {
		pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
		// Dilithium Signature Verification
		if !dilithium.Verify(signingHash[:], tx.Signature(), &pk) {
			return fmt.Errorf(errmsg.TXDilithiumSignatureVerificationFailed, "transfer", addrFrom, tx.Hash())
		}
	} else if len(tx.PK()) == xmss.ExtendedPKSize {
		// XMSS Signature Verification
		if !xmss.Verify(signingHash[:], tx.Signature(), misc.UnSizedXMSSPKToSizedPK(tx.PK())) {
			return fmt.Errorf(errmsg.TXXMSSSignatureVerificationFailed, "transfer", addrFrom, tx.Hash())
		}
	}

	return nil
}

func ValidateStakeTxn(tx *transactions.Stake, statedb *state.StateDB) error {
	txHash := tx.GenerateTxHash()

	addrFrom := tx.AddrFrom()
	accountState := statedb.GetOrNewStateObject(addrFrom)

	if tx.Hash() != txHash {
		return fmt.Errorf(errmsg.TXInvalidTxHash, "stake", tx.Hash(), addrFrom, txHash)
	}

	if tx.Nonce() != accountState.Nonce() {
		return fmt.Errorf(errmsg.TXInvalidNonce,
			"stake", txHash, addrFrom, tx.Nonce(), accountState.Nonce())
	}

	if tx.Amount() != 0 && tx.Amount() < config.GetDevConfig().StakeAmount {
		return fmt.Errorf(errmsg.TXInvalidStakeAmount,
			"stake", tx.Hash(), addrFrom, tx.Amount(), config.GetDevConfig().StakeAmount)
	}

	if len(tx.PK()) != dilithium.PKSizePacked {
		return fmt.Errorf(errmsg.TXInvalidDilithiumPKSize, "stake", tx.Hash(), addrFrom, len(tx.PK()))
	}

	balance := accountState.Balance()
	requiredBalance := tx.Gas()*tx.GasPrice() + tx.Amount()

	if balance.Cmp(big.NewInt(int64(requiredBalance))) < 0 {
		return fmt.Errorf(errmsg.TXInsufficientBalance,
			"stake", txHash, addrFrom, requiredBalance, balance)
	}

	// TODO: Remove this hardcoded 1000 gas
	gasRequired := uint64(1000)
	if tx.Gas() < gasRequired {
		return fmt.Errorf(errmsg.TXInsufficientGas,
			"stake", txHash, addrFrom, tx.Gas(), gasRequired)
	}

	if !dilithium.IsValidDilithiumAddress(tx.AddrFrom()) {
		return fmt.Errorf(errmsg.TXInvalidDilithiumAddrFrom, "stake", tx.Hash(), tx.AddrFrom())
	}

	pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
	signingHash := tx.GetSigningHash()

	if !dilithium.Verify(signingHash[:], tx.Signature(), &pk) {
		return fmt.Errorf(errmsg.TXDilithiumSignatureVerificationFailed, "stake", tx.Hash(), addrFrom)
	}

	return nil
}

func ValidateAttestTx(tx *transactions.Attest, statedb *state.StateDB, slotValidatorsMetaData *metadata.SlotValidatorsMetaData, partialBlockSigningHash common.Hash, slotNumber uint64, parentSlotNumber uint64) error {
	signedMessage := tx.GetSigningHash(partialBlockSigningHash)
	txHash := tx.TxHash(signedMessage)

	signerAddr := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(signerAddr)

	if tx.Hash() != txHash {
		return fmt.Errorf(errmsg.TXInvalidTxHash, "attest", tx.Hash(), signerAddr, txHash)
	}

	if tx.Nonce() != accountState.Nonce() {
		return fmt.Errorf(errmsg.TXInvalidNonce, "attest", tx.Hash(), signerAddr, tx.Nonce(), accountState.Nonce())
	}

	if len(tx.PK()) != dilithium.PKSizePacked {
		return fmt.Errorf(errmsg.TXInvalidDilithiumPKSize, "attest", tx.Hash(), signerAddr, len(tx.PK()))
	}

	pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
	if !dilithium.Verify(signedMessage[:], tx.Signature(), &pk) {
		return fmt.Errorf(errmsg.TXDilithiumSignatureVerificationFailed, "attest", tx.Hash(), signerAddr)
	}

	if !slotValidatorsMetaData.IsAttestor(misc.BytesToHexStr(tx.PK())) {
		return fmt.Errorf(errmsg.TXInvalidAttestor,
			"attest", misc.BytesToHexStr(txHash[:]), signerAddr, slotNumber)
	}

	parentEpoch := parentSlotNumber / config.GetDevConfig().BlocksPerEpoch
	currentEpoch := slotNumber / config.GetDevConfig().BlocksPerEpoch
	if accountState.StakeBalance().Uint64() < config.GetDevConfig().StakeAmount {
		/*
			This case happens, when the first block of epoch is created.
			During this case, the pendingStakeBalance is not yet added to the StakeBalance,
			and so we need to compare it with pendingStakeBalance, as that's the effective
			balance for this epoch
		*/
		if parentEpoch == currentEpoch {
			return fmt.Errorf(errmsg.TXInsufficientStakeBalance,
				"attest", tx.Hash(), signerAddr, accountState.StakeBalance())
		} else {
			if accountState.PendingStakeBalance().Uint64() < config.GetDevConfig().StakeAmount {
				return fmt.Errorf(errmsg.TXInsufficientStakeBalance,
					"attest", tx.Hash(), signerAddr, accountState.StakeBalance())
			}
		}
	}

	return nil
}

func ValidateCoinBaseTx(tx *transactions.CoinBase, statedb *state.StateDB, slotValidatorsMetaData *metadata.SlotValidatorsMetaData, blockSigningHash common.Hash, slotNumber, parentSlotNumber uint64, isGenesis bool) error {
	signedMessage := tx.GetSigningHash(blockSigningHash)
	txHash := tx.TxHash(signedMessage)

	signerAddr := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(signerAddr)

	if tx.Hash() != txHash {
		return fmt.Errorf(errmsg.TXInvalidTxHash, "coinbase", tx.Hash(), signerAddr, txHash)
	}

	if tx.Nonce() != accountState.Nonce() {
		return fmt.Errorf(errmsg.TXInvalidNonce, "coinbase", tx.Hash(), signerAddr, tx.Nonce(), accountState.Nonce())
	}

	if len(tx.PK()) != dilithium.PKSizePacked {
		return fmt.Errorf(errmsg.TXInvalidDilithiumPKSize, "coinbase", tx.Hash(), signerAddr, len(tx.PK()))
	}

	// Genesis block has unsigned coinbase txn
	if !isGenesis {
		pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
		if !dilithium.Verify(signedMessage[:], tx.Signature(), &pk) {
			return fmt.Errorf(errmsg.TXDilithiumSignatureVerificationFailed, "coinbase", tx.Hash(), signerAddr)
		}
	}

	validatorPK := tx.PK()
	if !slotValidatorsMetaData.IsSlotLeader(misc.BytesToHexStr(validatorPK[:])) {
		expectedSlotLeader := misc.GetAddressFromUnSizedPK(slotValidatorsMetaData.GetSlotLeaderPK())
		return fmt.Errorf(errmsg.TXInvalidSlotLeader,
			"coinbase", misc.BytesToHexStr(txHash[:]), signerAddr, slotNumber, expectedSlotLeader)
	}

	if tx.BlockProposerReward() != rewards.GetBlockReward() {
		return fmt.Errorf(errmsg.TXInvalidBlockReward,
			"coinbase", misc.BytesToHexStr(txHash[:]), signerAddr, tx.BlockProposerReward(), rewards.GetBlockReward())
	}

	if tx.AttestorReward() != rewards.GetAttestorReward() {
		return fmt.Errorf(errmsg.TXInvalidAttestorReward,
			"coinbase", misc.BytesToHexStr(txHash[:]), signerAddr, tx.AttestorReward(), rewards.GetAttestorReward())
	}

	// Genesis block proposer will have 0 stake balance initially
	if !isGenesis {
		parentEpoch := parentSlotNumber / config.GetDevConfig().BlocksPerEpoch
		currentEpoch := slotNumber / config.GetDevConfig().BlocksPerEpoch
		if accountState.StakeBalance().Uint64() < config.GetDevConfig().StakeAmount {
			/*
				This case happens, when the first block of epoch is created.
				During this case, the pendingStakeBalance is not yet added to the StakeBalance,
				and so we need to compare it with pendingStakeBalance, as that's the effective
				balance for this epoch
			*/
			if parentEpoch == currentEpoch {
				return fmt.Errorf(errmsg.TXInsufficientStakeBalance,
					"coinbase", tx.Hash(), signerAddr, accountState.StakeBalance())
			} else {
				if accountState.PendingStakeBalance().Uint64() < config.GetDevConfig().StakeAmount {
					return fmt.Errorf(errmsg.TXInsufficientStakeBalance,
						"coinbase", tx.Hash(), signerAddr, accountState.StakeBalance())
				}
			}
		}
	}

	// TODO: remove fee reward or the coinbase transaction
	//if tx.FeeReward() != stateContext.GetTotalTransactionFee() {
	//	log.Error("Invalid Fee Reward")
	//	log.Error("Expected Reward ", stateContext.GetTotalTransactionFee())
	//	log.Error("Found Reward ", tx.FeeReward())
	//	return false
	//}

	// TODO: provide total number of attestors for this check
	//balance := addressState.Balance()
	//if balance < tx.TotalAmounts() {
	//	log.Warn("Insufficient balance",
	//		"txhash", misc.BytesToHexStr(txHash),
	//		"balance", balance,
	//		"fee", tx.FeeReward())
	//	return false
	//}

	//ds := stateContext.GetDilithiumState(misc.BytesToHexStr(tx.PK()))
	//if ds == nil {
	//	log.Warn("Dilithium State not found for %s", misc.BytesToHexStr(tx.PK()))
	//	return false
	//}
	//if !ds.Stake() {
	//	log.Warn("Dilithium PK %s is not allowed to stake", misc.BytesToHexStr(tx.PK()))
	//	return false
	//}

	// TODO: Check the block proposer and attestor reward
	return nil
}

func ValidateTransaction(protoTx *protos.Transaction, statedb *state.StateDB) error {
	switch protoTx.Type.(type) {
	case *protos.Transaction_Stake:
		stakeTx := transactions.StakeTransactionFromPBData(protoTx)
		return ValidateStakeTxn(stakeTx, statedb)
	case *protos.Transaction_Transfer:
		transferTx := transactions.TransferTransactionFromPBData(protoTx)
		return ValidateTransferTxn(transferTx, statedb)
	default:
		return fmt.Errorf("unkown transaction type")
	}
}

func ValidateProtocolTransaction(protoTx *protos.ProtocolTransaction, statedb *state.StateDB, slotValidatorsMetaData *metadata.SlotValidatorsMetaData, hash common.Hash, slotNumber, parentSlotNumber uint64, isGenesis bool) error {
	switch protoTx.Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
		coinBaseTx := transactions.CoinBaseTransactionFromPBData(protoTx)
		return ValidateCoinBaseTx(coinBaseTx, statedb, slotValidatorsMetaData, hash, slotNumber, parentSlotNumber, isGenesis)
	case *protos.ProtocolTransaction_Attest:
		attestTx := transactions.AttestTransactionFromPBData(protoTx)
		return ValidateAttestTx(attestTx, statedb, slotValidatorsMetaData, hash, slotNumber, parentSlotNumber)
	default:
		return fmt.Errorf("unkown transaction type")
	}
}

func ProcessEpochMetaData(b *block.Block, statedb *state.StateDB, epochMetaData *metadata.EpochMetaData) error {
	for _, pbData := range b.ProtocolTransactions() {
		address := misc.GetDilithiumAddressFromUnSizedPK(pbData.GetPk())
		validatorStakeBalance := statedb.GetStakeBalance(address).Uint64()
		requiredStakeAmount := config.GetDevConfig().StakeAmount
		if validatorStakeBalance < requiredStakeAmount && b.SlotNumber() != 0 {
			return fmt.Errorf("balance not loaded for the validator %s", address)
		}
		epochMetaData.AddTotalStakeAmountFound(requiredStakeAmount)
	}
	return nil
}

func UpdateStakeValidators(statedb *state.StateDB, pendingStakeValidatorsUpdate [][]byte, epochMetaData *metadata.EpochMetaData) error {
	for _, validatorPK := range pendingStakeValidatorsUpdate {
		account := statedb.GetOrNewStateObject(misc.GetAddressFromUnSizedPK(validatorPK))
		if account.PendingStakeBalance().Uint64() == 0 {
			epochMetaData.RemoveValidators(validatorPK)

			account.AddBalance(account.StakeBalance())
			account.SetStakeBalance(common.Big0)
			log.Info("Removed validator ", misc.GetAddressFromUnSizedPK(validatorPK))
		} else {
			account.AddBalance(account.StakeBalance())
			account.SetStakeBalance(account.PendingStakeBalance())
			account.SetPendingStakeBalance(common.Big0)

			epochMetaData.AddValidators(validatorPK)
			log.Info("Added validator ", misc.GetAddressFromUnSizedPK(validatorPK))
		}
	}

	return nil
}

func receiptsToByteReceipts(receipts types.Receipts) ([]byte, error) {
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
	}
	return rlp.EncodeToBytes(storageReceipts)
}

func Re(receipts types.Receipts) ([]byte, error) {
	return receiptsToByteReceipts(receipts)
}

func TransferTxAsMessage(tx *transactions.Transfer, baseFee *big.Int) (types.Message, error) {
	bigIntGasPrice := big.NewInt(int64(tx.GasPrice()))
	bigIntGasFeeCap := big.NewInt(int64(tx.GasFeeCap()))
	bigIntGasTipCap := big.NewInt(int64(tx.GasTipCap()))
	bigIntValue := big.NewInt(int64(tx.Value()))
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		bigIntGasPrice = math.BigMin(bigIntGasPrice.Add(bigIntGasTipCap, baseFee), bigIntGasFeeCap)
	}

	msg := types.NewMessage(tx.AddrFrom(), tx.To(), tx.Nonce(), bigIntValue, tx.Gas(), bigIntGasPrice, bigIntGasFeeCap, bigIntGasTipCap, tx.Data(), nil, false)

	return msg, nil
}

// GetRandomSeed has temporary code for random seed calculation, this will be updated before final release
func GetRandomSeed(hash common.Hash) int64 {
	h := md5.New()
	h.Write(hash[:])
	randomSeed := int64(binary.BigEndian.Uint64(h.Sum(nil)))
	return randomSeed
}
