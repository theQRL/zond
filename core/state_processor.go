package core

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/block/rewards"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/core/state"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/core/vm"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/params"
	"github.com/theQRL/zond/protos"
	state2 "github.com/theQRL/zond/state"
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

	if !reflect.DeepEqual(blockProposerDilithiumAddress, config.GetDevConfig().Genesis.FoundationDilithiumAddress) {
		expectedFoundationDilithiumAddress := hex.EncodeToString(config.GetDevConfig().Genesis.FoundationDilithiumAddress[:])
		foundFoundationDilithiumAddress := hex.EncodeToString(config.GetDevConfig().Genesis.FoundationDilithiumAddress[:])

		log.Warn("block proposer dilithium address is not matching with the foundation dilithium address in config")
		log.Warn("expected foundation dilithium address ", expectedFoundationDilithiumAddress)
		log.Warn("found foundation dilithium address ", foundFoundationDilithiumAddress)
		fmt.Println(blockProposerDilithiumAddress == config.GetDevConfig().Genesis.FoundationDilithiumAddress)
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
	validatorsType := make(map[string]uint8)

	blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()
	validatorsType[hex.EncodeToString(blockProposerDilithiumPK)] = 1

	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = b.Header()
		blockHash   = b.Hash()
		blockNumber = big.NewInt(int64(b.Number()))
		allLogs     []*types.Log
		gp          = new(GasPool).AddGas(b.GasLimit())
	)

	blockContext := NewEVMBlockContext(header, p.getHashFunc, b.Minter(), b.Timestamp())
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

			if !ValidateCoinBaseTx(coinBaseTx, statedb, validatorsType, b.BlockSigningHash(), false) {
				return nil, nil, 0, fmt.Errorf("coinbase tx validation failed")
			}

			coinBaseTxHash := coinBaseTx.TxHash(coinBaseTx.GetSigningHash(b.BlockSigningHash()))
			statedb.Prepare(coinBaseTxHash, 0)
			receipt, err = applyCoinBaseTransaction(statedb, stateContext, blockNumber, blockHash, coinBaseTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply coinbase tx %d [%v]: %w", 0, coinBaseTxHash, err)
			}

		case *protos.ProtocolTransaction_Attest:
			attestTx := transactions.AttestTransactionFromPBData(protoTx)

			if !ValidateAttestTx(attestTx, statedb, validatorsType, b.PartialBlockSigningHash()) {
				return nil, nil, 0, fmt.Errorf("attest tx validation failed")
			}

			txHash := attestTx.TxHash(attestTx.GetSigningHash(b.PartialBlockSigningHash()))
			statedb.Prepare(txHash, i)
			receipt, err = applyAttestTransaction(statedb, stateContext, blockNumber, blockHash, attestTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply attest tx %d [%v]: %w", 0, txHash, err)
			}
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	validatorsStakeAmount := make(map[string]uint64)

	// Iterate over and process the individual transactions
	for i, protoTx := range b.Transactions() {
		tx := transactions.ProtoToTransaction(protoTx)
		statedb.Prepare(tx.Hash(), i)

		var receipt *types.Receipt
		var err error

		switch protoTx.Type.(type) {
		case *protos.Transaction_Stake:
			stakeTx := transactions.StakeTransactionFromPBData(protoTx)

			strDilithiumPK := hex.EncodeToString(stakeTx.PK())
			validatorsStakeAmount[strDilithiumPK] = config.GetDevConfig().StakeAmount

			if !ValidateStakeTxn(stakeTx, statedb) {
				return nil, nil, 0, fmt.Errorf("stake tx validation failed")
			}
			receipt, err = applyStakeTransaction(gp, statedb, blockNumber, blockHash, stakeTx, usedGas, b.Minter())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply stake tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}

		case *protos.Transaction_Transfer:
			transferTx := transactions.TransferTransactionFromPBData(protoTx)
			if !ValidateTransferTxn(transferTx, statedb) {
				return nil, nil, 0, fmt.Errorf("transfer tx validation failed")
			}
			msg, err := tx.AsMessage(header.BaseFee())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
			receipt, err = applyTransaction(msg, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply transfer tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)

		// If code size is 0 in that case it is an account and not a contract
		if statedb.GetCodeSize(tx.AddrFrom()) == 0 && xmss.IsValidXMSSAddress(tx.AddrFrom()) {
			statedb.SetOTSBitfield(tx.AddrFrom(), misc.GetOTSIndexFromSignature(tx.Signature()), false)
		}

	}

	epochMetaData := stateContext.GetEpochMetaData()
	validatorsStateChanged := make(map[string]bool)
	err := b.ProcessEpochMetaData(epochMetaData, validatorsStakeAmount, validatorsStateChanged)
	if err != nil {
		log.Error("Failed to Process Epoch MetaData")
		return nil, nil, 0, err
	}

	// For Genesis Block total stake found and alloted must be same
	epochMetaData.UpdatePrevEpochStakeData(epochMetaData.TotalStakeAmountFound(),
		epochMetaData.TotalStakeAmountFound())

	var randomSeed int64
	h := md5.New()
	pHash := b.ParentHash()
	h.Write(pHash[:])
	randomSeed = int64(binary.BigEndian.Uint64(h.Sum(nil)))

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

	bytesBlock, err := b.Serialize()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to serialize block : %w", err)
	}
	err = stateContext.Commit(block.GetBlockStorageKey(b.Hash()), bytesBlock, trieRoot, true)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit statecontext : %w", err)
	}

	return receipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) Process(b *block.Block, statedb *state.StateDB, stateContext *state2.StateContext, validatorsType map[string]uint8, isFinalizedState bool, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = b.Header()
		blockHash   = b.Hash()
		blockNumber = big.NewInt(int64(b.Number()))
		allLogs     []*types.Log
		gp          = new(GasPool).AddGas(b.GasLimit())
	)

	blockContext := NewEVMBlockContext(header, p.getHashFunc, b.Minter(), b.Timestamp())
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

			if !ValidateCoinBaseTx(coinBaseTx, statedb, validatorsType, b.BlockSigningHash(), false) {
				return nil, nil, 0, fmt.Errorf("coinbase tx validation failed")
			}

			coinBaseTxHash := coinBaseTx.TxHash(coinBaseTx.GetSigningHash(b.BlockSigningHash()))
			statedb.Prepare(coinBaseTxHash, 0)
			receipt, err = applyCoinBaseTransaction(statedb, stateContext, blockNumber, blockHash, coinBaseTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply coinbase tx %d [%v]: %w", 0, coinBaseTxHash, err)
			}

		case *protos.ProtocolTransaction_Attest:
			attestTx := transactions.AttestTransactionFromPBData(protoTx)

			if !ValidateAttestTx(attestTx, statedb, validatorsType, b.PartialBlockSigningHash()) {
				return nil, nil, 0, fmt.Errorf("attest tx validation failed")
			}

			txHash := attestTx.TxHash(attestTx.GetSigningHash(b.PartialBlockSigningHash()))
			statedb.Prepare(txHash, i)
			receipt, err = applyAttestTransaction(statedb, stateContext, blockNumber, blockHash, attestTx)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply attest tx %d [%v]: %w", 0, txHash, err)
			}
		}
		receipts = append(receipts, receipt)
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
			if !ValidateStakeTxn(stakeTx, statedb) {
				return nil, nil, 0, fmt.Errorf("stake tx validation failed")
			}
			receipt, err = applyStakeTransaction(gp, statedb, blockNumber, blockHash, stakeTx, usedGas, b.Minter())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply stake tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}

		case *protos.Transaction_Transfer:
			transferTx := transactions.TransferTransactionFromPBData(protoTx)
			if !ValidateTransferTxn(transferTx, statedb) {
				return nil, nil, 0, fmt.Errorf("transfer tx validation failed")
			}
			msg, err := tx.AsMessage(header.BaseFee())
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
			receipt, err = applyTransaction(msg, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("could not apply transfer tx %d [%v]: %w", i, tx.Hash().Hex(), err)
			}
		}
		receipts = append(receipts, receipt)
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

	bytesBlock, err := b.Serialize()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to serialize block : %w", err)
	}
	err = stateContext.Commit(block.GetBlockStorageKey(b.Hash()), bytesBlock, trieRoot, isFinalizedState)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to commit statecontext : %w", err)
	}

	return receipts, allLogs, *usedGas, nil
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
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
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

	accountState.SetNonce(tx.Nonce())

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
	accountState.AddBalance(big.NewInt(int64(tx.Gas()-(*usedGas)) * int64(tx.GasPrice())))
	// add gas fee to the block proposer address
	statedb.AddBalance(*minter, big.NewInt(int64((*usedGas)*tx.GasPrice())))

	if err := gp.SubGas(StakeTxGas); err != nil {
		return nil, err
	}
	/* -- Stake Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
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
		return nil, fmt.Errorf("failed to process block proposer %s | Reason: %w", hex.EncodeToString(tx.PK()), err)
	}

	accountState.SetNonce(tx.Nonce())

	validatorsFlag := stateContext.ValidatorsFlag()

	strBlockProposerDilithiumPK := hex.EncodeToString(stateContext.BlockProposer())

	for strValidatorDilithiumPK, _ := range validatorsFlag {
		validatorDilithiumPK, err := hex.DecodeString(strValidatorDilithiumPK)
		if err != nil {
			return nil, err
		}
		accountState := statedb.GetOrNewStateObject(misc.GetDilithiumAddressFromUnSizedPK(validatorDilithiumPK))

		if strValidatorDilithiumPK == strBlockProposerDilithiumPK {
			accountState.AddBalance(big.NewInt(int64(tx.BlockProposerReward())))
			accountState.AddBalance(big.NewInt(int64(tx.FeeReward())))
		} else {
			accountState.AddBalance(big.NewInt(int64(tx.AttestorReward())))
		}
	}

	coinBaseAccountState := statedb.GetOrNewStateObject(config.GetDevConfig().Genesis.CoinBaseAddress)
	// len(validatorsFlag) - 1 is the number of attestors, as one of the validator is block proposer
	coinBaseAccountState.SubBalance(big.NewInt(int64(tx.TotalRewardExceptFeeReward(uint64(len(validatorsFlag) - 1)))))

	signedMessage := tx.GetSigningHash(stateContext.BlockSigningHash())
	txHash := tx.TxHash(signedMessage)
	/* -- Coinbase Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root}
	receipt.Status = types.ReceiptStatusSuccessful

	receipt.TxHash = txHash

	// Set the receipt logs and create the bloom filter.
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, nil
}

func applyAttestTransaction(statedb *state.StateDB, stateContext *state2.StateContext, blockNumber *big.Int, blockHash common.Hash, tx *transactions.Attest) (*types.Receipt, error) {
	// Update the state with pending changes.
	var root []byte

	statedb.Finalise(true)

	/* -- Attest Transaction changes starts -- */
	signedMessage := tx.GetSigningHash(stateContext.PartialBlockSigningHash())
	txHash := tx.TxHash(signedMessage)

	address := misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	accountState := statedb.GetOrNewStateObject(address)

	if err := stateContext.ProcessAttestorsFlag(tx.PK(), accountState.StakeBalance()); err != nil {
		return nil, fmt.Errorf("failed to process attest transaction for attestor  %s | Reason: %w", hex.EncodeToString(tx.PK()), err)
	}
	/* -- Attest Transaction changes ends -- */

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root}
	receipt.Status = types.ReceiptStatusSuccessful

	receipt.TxHash = txHash

	// Set the receipt logs and create the bloom filter.
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, nil
}

func ValidateTransferTxn(tx *transactions.Transfer, statedb *state.StateDB) bool {
	txHash := tx.Hash()

	addrFrom := tx.AddrFrom()
	addressState := statedb.GetOrNewStateObject(addrFrom)

	if tx.Nonce() != addressState.Nonce() {
		log.Warn(fmt.Sprintf("Transfer [%s] Invalid Nonce %d, Expected Nonce %d",
			hex.EncodeToString(txHash[:]), tx.Nonce(), addressState.Nonce()))
		return false
	}

	if len(tx.PK()) == xmss.ExtendedPKSize {
		if misc.IsUsedOTSIndex(tx.OTSIndex(), addressState.OTSBitfield()) {
			log.Warn(fmt.Sprintf("Transfer [%s] OTS Index %d is already used",
				hex.EncodeToString(txHash[:]), tx.OTSIndex()))
			return false
		}
	}

	balance := addressState.Balance().Uint64()
	if balance < tx.Value()+tx.Gas()*tx.GasPrice() {
		log.Warn("Insufficient balance",
			"txhash", hex.EncodeToString(txHash[:]),
			"balance", balance,
			"value", tx.Value(),
			"max fee", tx.Gas()*tx.GasPrice())
		return false
	}

	// TODO: Move to some common validation
	if !(xmss.IsValidXMSSAddress(tx.AddrFrom()) || dilithium.IsValidDilithiumAddress(tx.AddrFrom())) {
		log.Warn("[Transfer] Invalid address addr_from: %s", tx.AddrFrom())
		return false
	}

	if len(tx.To()) != 0 && !(xmss.IsValidXMSSAddress(*tx.To()) || dilithium.IsValidDilithiumAddress(*tx.To())) {
		log.Warn("[Transfer] Invalid address addr_to: %s", tx.To())
		return false
	}

	if reflect.DeepEqual(tx.AddrFrom(), config.GetDevConfig().Genesis.CoinBaseAddress) {
		log.Warn("from address cannot be a coinbase address")
		return false
	}

	if reflect.DeepEqual(*tx.To(), config.GetDevConfig().Genesis.CoinBaseAddress) {
		log.Warn("to address cannot be a coinbase address")
		return false
	}

	dataLen := len(tx.Data())
	// TODO: Move the hardcoded value to config
	if dataLen > 24*1024 {
		log.Warn("Data length beyond limit: %d", dataLen)
		return false
	}

	signingHash := tx.GetSigningHash()
	if len(tx.PK()) == dilithium.PKSizePacked {
		pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
		// Dilithium Signature Verification
		if !dilithium.Verify(signingHash[:], tx.Signature(), &pk) {
			log.Warn("Dilithium Signature Verification Failed")
			return false
		}
	} else if len(tx.PK()) == xmss.ExtendedPKSize {
		// XMSS Signature Verification
		if !xmss.Verify(signingHash[:], tx.Signature(), misc.UnSizedXMSSPKToSizedPK(tx.PK())) {
			log.Warn("XMSS Verification Failed")
			return false
		}
	}

	return true
}

func ValidateStakeTxn(tx *transactions.Stake, statedb *state.StateDB) bool {
	txHash := tx.Hash()

	addrFrom := tx.AddrFrom()

	accountState := statedb.GetOrNewStateObject(addrFrom)

	if tx.Nonce() != accountState.Nonce() {
		log.Warn(fmt.Sprintf("Stake [%s] Invalid Nonce %d, Expected Nonce %d",
			hex.EncodeToString(txHash[:]), tx.Nonce(), accountState.Nonce()))
		return false
	}

	if tx.Amount() < config.GetDevConfig().StakeAmount {
		log.Warn("Invalid stake amount",
			"Must be multiplier of", config.GetDevConfig().StakeAmount,
			"Stake Amount", tx.Amount())
	}

	balance := accountState.Balance()
	requiredBalance := tx.Gas()*tx.GasPrice() + tx.Amount()

	if balance.Cmp(big.NewInt(int64(requiredBalance))) < 0 {
		log.Warn("Insufficient balance",
			"txhash", hex.EncodeToString(txHash[:]),
			"balance", balance,
			"required balance", requiredBalance)
		return false
	}

	// TODO: Remove this hardcoded 1000 gas
	if tx.Gas() < 1000 {
		log.Warn("Insufficient stake transaction gas",
			"txhash", hex.EncodeToString(txHash[:]),
			"min stake txn gas required", 1000,
			"gas provided", tx.Gas())
		return false
	}

	if !dilithium.IsValidDilithiumAddress(tx.AddrFrom()) {
		log.Warn("[Stake] Invalid dilithium address in addr_from: %s", tx.AddrFrom())
		return false
	}

	pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
	signingHash := tx.GetSigningHash()
	// Dilithium Signature Verification
	if !dilithium.Verify(signingHash[:], tx.Signature(), &pk) {
		log.Warn("Dilithium Signature Verification Failed")
		return false
	}

	return true
}

func ValidateAttestTx(tx *transactions.Attest, statedb *state.StateDB, validatorsType map[string]uint8, partialBlockSigningHash common.Hash) bool {
	signedMessage := tx.GetSigningHash(partialBlockSigningHash)
	txHash := tx.TxHash(signedMessage)

	if len(tx.PK()) != dilithium.PKSizePacked {
		log.Error("Invalid Dilithium PK ")
		return false
	}

	pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
	if !dilithium.Verify(signedMessage[:], tx.Signature(), &pk) {
		log.Warn(fmt.Sprintf("Dilithium Signature Verification failed for Attest Txn %s",
			hex.EncodeToString(txHash[:])))
		return false
	}

	validatorPK := tx.PK()
	validatorType, ok := validatorsType[hex.EncodeToString(validatorPK[:])]
	if !ok || validatorType != 0 {
		log.Error("invalid attestor",
			"TxHash", hex.EncodeToString(txHash[:]))
		return false
	}

	accountState := statedb.GetOrNewStateObject(misc.GetDilithiumAddressFromUnSizedPK(tx.PK()))
	if accountState.StakeBalance().Uint64() < config.GetDevConfig().StakeAmount {
		return false
	}

	return true
}

func ValidateCoinBaseTx(tx *transactions.CoinBase, statedb *state.StateDB, validatorsType map[string]uint8, blockSigningHash common.Hash, isGenesis bool) bool {
	signedMessage := tx.GetSigningHash(blockSigningHash)
	txHash := tx.TxHash(signedMessage)

	// Genesis block has unsigned coinbase txn
	if !isGenesis {
		pk := misc.UnSizedDilithiumPKToSizedPK(tx.PK())
		if !dilithium.Verify(signedMessage[:], tx.Signature(), &pk) {
			log.Warn(fmt.Sprintf("Dilithium Signature Verification failed for CoinBase Txn %s",
				hex.EncodeToString(txHash[:])))
			return false
		}
	}

	if len(tx.PK()) != dilithium.PKSizePacked {
		log.Error("Invalid Dilithium PK",
			"TxHash", hex.EncodeToString(txHash[:]))
		return false
	}

	validatorPK := tx.PK()
	validatorType, ok := validatorsType[hex.EncodeToString(validatorPK[:])]
	if !ok || validatorType != 1 {
		log.Error("invalid block proposer",
			"TxHash", hex.EncodeToString(txHash[:]))
		return false
	}

	coinBaseAddress := config.GetDevConfig().Genesis.CoinBaseAddress
	coinBaseAccountState := statedb.GetOrNewStateObject(coinBaseAddress)

	if tx.Nonce() != coinBaseAccountState.Nonce() {
		log.Warn(fmt.Sprintf("CoinBase [%s] Invalid Nonce %d, Expected Nonce %d",
			hex.EncodeToString(txHash[:]), tx.Nonce(), coinBaseAccountState.Nonce()))
		return false
	}

	if tx.BlockProposerReward() != rewards.GetBlockReward() {
		log.Error("Invalid Block Proposer Reward")
		log.Error("Expected Reward ", rewards.GetBlockReward())
		log.Error("Found Reward ", tx.BlockProposerReward())
		return false
	}

	if tx.AttestorReward() != rewards.GetAttestorReward() {
		log.Error("Invalid Attestor Reward")
		log.Error("Expected Reward ", rewards.GetAttestorReward())
		log.Error("Found Reward ", tx.AttestorReward())
		return false
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
	//		"txhash", hex.EncodeToString(txHash),
	//		"balance", balance,
	//		"fee", tx.FeeReward())
	//	return false
	//}

	//ds := stateContext.GetDilithiumState(hex.EncodeToString(tx.PK()))
	//if ds == nil {
	//	log.Warn("Dilithium State not found for %s", hex.EncodeToString(tx.PK()))
	//	return false
	//}
	//if !ds.Stake() {
	//	log.Warn("Dilithium PK %s is not allowed to stake", hex.EncodeToString(tx.PK()))
	//	return false
	//}

	// TODO: Check the block proposer and attestor reward
	return true
}

func ValidateTransaction(protoTx *protos.Transaction, statedb *state.StateDB) bool {
	switch protoTx.Type.(type) {
	case *protos.Transaction_Stake:
		stakeTx := transactions.StakeTransactionFromPBData(protoTx)
		return ValidateStakeTxn(stakeTx, statedb)
	case *protos.Transaction_Transfer:
		transferTx := transactions.TransferTransactionFromPBData(protoTx)
		return ValidateTransferTxn(transferTx, statedb)
	default:
		panic("Unknown txn")
	}
	return false
}

func ValidateProtocolTransaction(protoTx *protos.ProtocolTransaction, statedb *state.StateDB, validatorsType map[string]uint8, hash common.Hash, isGenesis bool) bool {
	switch protoTx.Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
		coinBaseTx := transactions.CoinBaseTransactionFromPBData(protoTx)
		return ValidateCoinBaseTx(coinBaseTx, statedb, validatorsType, hash, isGenesis)
	case *protos.ProtocolTransaction_Attest:
		attestTx := transactions.AttestTransactionFromPBData(protoTx)
		return ValidateAttestTx(attestTx, statedb, validatorsType, hash)
	default:
		panic("Unknown txn")
	}
	return false
}
