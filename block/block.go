package block

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/zond/block/rewards"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"github.com/theQRL/zond/transactions"
	"math/big"
	"reflect"
)

type Header struct {
	pbData *protos.BlockHeader
}

func HeaderFromPBData(header *protos.BlockHeader) *Header {
	return &Header{
		pbData: header,
	}
}

func (h *Header) Number() *big.Int {
	return big.NewInt(int64(h.pbData.SlotNumber))
}

func (h *Header) ParentHash() common.Hash {
	var output common.Hash
	copy(output[:], h.pbData.ParentHash)
	return output
}

func (h *Header) BaseFee() *big.Int {
	return big.NewInt(int64(h.pbData.BaseFee))
}

func (h *Header) GasLimit() uint64 {
	return h.pbData.GasLimit
}

func (h *Header) GasUsed() *big.Int {
	return big.NewInt(int64(h.pbData.GasUsed))
}

type Block struct {
	header *Header
	pbData *protos.Block
}

func (b *Block) Header() *Header {
	return b.header
}

func (b *Block) Timestamp() uint64 {
	return b.pbData.Header.TimestampSeconds
}

func (b *Block) ParentHash() common.Hash {
	return b.header.ParentHash()
}

func (b *Block) Epoch() uint64 {
	return b.pbData.Header.SlotNumber / config.GetDevConfig().BlocksPerEpoch
}

func (b *Block) SlotNumber() uint64 {
	return b.pbData.Header.SlotNumber
}

func (b *Block) Number() uint64 {
	return b.pbData.Header.SlotNumber
}

func (b *Block) GasLimit() uint64 {
	return b.pbData.Header.GasLimit
}

func (b *Block) Minter() *common.Address {
	// TODO: Fix Minter
	// b.pbData.ProtocolTransactions[0].GetPk()
	// get xmss address by dilithium pk
	return &common.Address{}
}

func (b *Block) Hash() common.Hash {
	blockSigningHash := b.BlockSigningHash()
	tmp := new(bytes.Buffer)
	tmp.Write(blockSigningHash[:])
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	txHash := coinBaseTx.TxHash(coinBaseTx.GetSigningHash(blockSigningHash))
	tmp.Write(txHash[:])

	headerHash := sha256.New()
	headerHash.Write(tmp.Bytes())
	hash := headerHash.Sum(nil)

	var output common.Hash
	copy(output[:], hash)
	return output
}

func (b *Block) Transactions() []*protos.Transaction {
	return b.pbData.Transactions
}

func (b *Block) ProtocolTransactions() []*protos.ProtocolTransaction {
	return b.pbData.ProtocolTransactions
}

func (b *Block) PBData() *protos.Block {
	return b.pbData
}

func (b *Block) Serialize() ([]byte, error) {
	return proto.Marshal(b.pbData)
}

func (b *Block) DeSerialize(data []byte) error {
	b.pbData = &protos.Block{}
	b.header = &Header{}

	if err := proto.Unmarshal(data, b.pbData); err != nil {
		return err
	}

	b.Header().pbData = b.pbData.Header
	return nil
}

func (b *Block) PartialBlockSigningHash() common.Hash {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction

	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, b.Timestamp())
	binary.Write(tmp, binary.BigEndian, b.Header().Number().Uint64())
	pHash := b.Header().ParentHash()
	tmp.Write(pHash[:])

	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		txHash := tx.Hash()
		tmp.Write(txHash[:])
	}
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	unsignedHash := coinBaseTx.GetUnsignedHash()
	tmp.Write(unsignedHash[:])

	h := sha256.New()
	h.Write(tmp.Bytes())

	var hash common.Hash
	outputHash := h.Sum(nil)
	copy(hash[:], outputHash)

	return hash
}

func (b *Block) BlockSigningHash() common.Hash {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction

	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, b.Timestamp())
	binary.Write(tmp, binary.BigEndian, b.Header().Number().Uint64())
	binary.Write(tmp, binary.BigEndian, b.Header().BaseFee())
	binary.Write(tmp, binary.BigEndian, b.Header().GasLimit())
	binary.Write(tmp, binary.BigEndian, b.Header().GasUsed())

	pHash := b.Header().ParentHash()
	tmp.Write(pHash[:])

	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		txHash := tx.Hash()
		tmp.Write(txHash[:])
	}
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	unsignedHash := coinBaseTx.GetUnsignedHash()
	tmp.Write(unsignedHash[:])
	for i := 1; i < len(b.ProtocolTransactions()); i++ {
		tx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[i])
		txHash := tx.TxHash(tx.GetSigningHash(b.PartialBlockSigningHash()))
		tmp.Write(txHash[:])
	}

	h := sha256.New()
	h.Write(tmp.Bytes())

	var hash common.Hash
	outputHash := h.Sum(nil)
	copy(hash[:], outputHash)
	return hash
}

func (b *Block) Attest(networkID uint64, d *dilithium.Dilithium) (*transactions.Attest, error) {
	attestTx := transactions.NewAttest(networkID, b.ProtocolTransactions()[0].Nonce)
	signingHash := attestTx.GetSigningHash(b.PartialBlockSigningHash())
	attestTx.Sign(d, signingHash[:])
	return attestTx, nil
}

func (b *Block) AddAttestTx(attestTx *transactions.Attest) {
	partialBlockSigningHash := b.PartialBlockSigningHash()
	attestTxHash := attestTx.TxHash(attestTx.GetSigningHash(partialBlockSigningHash))
	for _, protoTX := range b.ProtocolTransactions()[1:] {
		tx := transactions.ProtoToProtocolTransaction(protoTX)
		if reflect.DeepEqual(tx.TxHash(tx.GetSigningHash(partialBlockSigningHash)),
			attestTxHash) {
			return
		}
	}
	b.pbData.ProtocolTransactions = append(b.ProtocolTransactions(), attestTx.PBData())
}

func (b *Block) SignByProposer(d *dilithium.Dilithium) {
	coinbaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	message := coinbaseTx.GetSigningHash(b.BlockSigningHash())
	coinbaseTx.Sign(d, message[:])
	b.ProtocolTransactions()[0] = coinbaseTx.PBData()
}

func (b *Block) ProcessEpochMetaData(epochMetaData *metadata.EpochMetaData,
	validatorsStakeAmount map[string]uint64, validatorsStateChanged map[string]bool) error {

	for _, pbData := range b.Transactions() {
		switch pbData.Type.(type) {
		case *protos.Transaction_Stake:
			// TODO: Need to add this address into the epoch metadata, if it doesn't exist
			// add existing stake amount to balance
			// then set stake amount equal to tx.Amount
			// finally set the pendingStakeBalance to 0 for all the validators

			tx := transactions.ProtoToTransaction(pbData)
			if pbData.GetStake().Amount > 0 {
				epochMetaData.AddValidators(tx.PK())
				validatorsStateChanged[hex.EncodeToString(tx.PK())] = true
			} else {
				epochMetaData.RemoveValidators(tx.PK())
				validatorsStateChanged[hex.EncodeToString(tx.PK())] = false
			}
		}
	}
	for _, pbData := range b.ProtocolTransactions() {
		strPK := hex.EncodeToString(pbData.Pk)
		amount, ok := validatorsStakeAmount[strPK]
		if !ok {
			return errors.New(fmt.Sprintf("balance not loaded for the validator %s", strPK))
		}
		epochMetaData.AddTotalStakeAmountFound(amount)
	}
	return nil
}

func NewBlock(networkId uint64, timestamp uint64, proposerDilithiumPK []byte, slotNumber uint64,
	parentHeaderHash common.Hash, txs []*protos.Transaction, protocolTxs []*protos.ProtocolTransaction,
	lastCoinBaseNonce uint64) *Block {
	b := &Block{
		pbData: &protos.Block{},
	}

	blockHeader := &protos.BlockHeader{}
	blockHeader.TimestampSeconds = timestamp
	blockHeader.SlotNumber = slotNumber
	blockHeader.ParentHash = parentHeaderHash[:]
	blockHeader.GasLimit = config.GetDevConfig().BlockGasLimit

	b.pbData.Header = blockHeader
	b.header = &Header{blockHeader}

	feeReward := uint64(0)
	for _, tx := range txs {
		b.pbData.Transactions = append(b.pbData.Transactions, tx)
		feeReward += tx.Gas * tx.GasPrice
	}

	blockReward := rewards.GetBlockReward()
	attestorReward := rewards.GetAttestorReward()
	coinBase := transactions.NewCoinBase(networkId, proposerDilithiumPK, blockReward,
		attestorReward, feeReward, lastCoinBaseNonce)

	b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, coinBase.PBData())

	for _, tx := range protocolTxs {
		b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, tx)
	}

	return b
}

func (b *Block) UpdateFinalizedEpoch(db *db.DB, stateContext *state.StateContext) error {
	currentEpochMetaData := stateContext.GetEpochMetaData()
	// Ignore Finalization if TotalStakeAmountFound is less than the 2/3rd of TotalStakeAmountAlloted
	if currentEpochMetaData.TotalStakeAmountFound()*3 < currentEpochMetaData.TotalStakeAmountAlloted()*2 {
		return nil
	}

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	currentEpoch := b.Epoch()
	mainChainMetaData := stateContext.GetMainChainMetaData()
	finalizedBlockEpoch := mainChainMetaData.FinalizedBlockSlotNumber() / blocksPerEpoch

	if mainChainMetaData.FinalizedBlockSlotNumber() == 0 {
		if currentEpoch-finalizedBlockEpoch < 3 {
			return nil
		}
	} else if currentEpoch-finalizedBlockEpoch <= 3 {
		return nil
	}

	bm, err := metadata.GetBlockMetaData(db, b.ParentHash())
	if err != nil {
		log.Error("[UpdateFinalizedEpoch] Failed to GetBlockMetaData")
		return err
	}

	// Skip finalization if epoch is 0
	if bm.Epoch() == 0 {
		return nil
	}

	for {
		newBM, err := metadata.GetBlockMetaData(db, bm.ParentHeaderHash())
		if err != nil {
			log.Error("[UpdateFinalizedEpoch] Failed to GetBlockMetaData")
			return err
		}
		if bm.Epoch() != newBM.Epoch() {
			break
		}
		bm = newBM
	}

	// Skip finalization if second last epoch is 0
	if bm.Epoch() == 0 {
		return nil
	}

	epochMetaData, err := metadata.GetEpochMetaData(db, bm.SlotNumber(), bm.ParentHeaderHash())
	if err != nil {
		log.Error("[UpdateFinalizedEpoch] Failed to load EpochMetaData for ", bm.Epoch()-1)
		return err
	}
	if epochMetaData.TotalStakeAmountFound()*3 < epochMetaData.TotalStakeAmountAlloted()*2 {
		return nil
	}

	for {
		newBM, err := metadata.GetBlockMetaData(db, bm.ParentHeaderHash())
		if err != nil {
			log.Error("[UpdateFinalizedEpoch] Failed to GetBlockMetaData")
			return err
		}
		if bm.Epoch() != newBM.Epoch() {
			break
		}
		bm = newBM
	}

	headerHash := bm.ParentHeaderHash()
	blockMetaDataPathForFinalization := make([]*metadata.BlockMetaData, 0)
	for {
		bm, err := metadata.GetBlockMetaData(db, headerHash)
		if err != nil {
			log.Error("[UpdateFinalizedEpoch] Failed To Load GetBlockMetaData for ", hex.EncodeToString(headerHash[:]))
			return err
		}
		if reflect.DeepEqual(bm.HeaderHash(), stateContext.GetMainChainMetaData().FinalizedBlockHeaderHash()) {
			break
		}
		blockMetaDataPathForFinalization = append(blockMetaDataPathForFinalization, bm)
		headerHash = bm.ParentHeaderHash()
	}

	return stateContext.Finalize(blockMetaDataPathForFinalization)
}

func CalculateEpochMetaData(db *db.DB, slotNumber uint64,
	parentHeaderHash common.Hash, parentSlotNumber uint64) (*metadata.EpochMetaData, error) {

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

	validatorsStakeAmount := make(map[string]uint64)
	totalStakeAmountAlloted := uint64(len(epochMetaData.Validators())) * config.GetDevConfig().StakeAmount
	epochMetaData.UpdatePrevEpochStakeData(0,
		totalStakeAmountAlloted)

	validatorsStateChanged := make(map[string]bool)
	for i := lenPathToFirstBlockOfEpoch - 1; i >= 0; i-- {
		b, err := GetBlock(db, pathToFirstBlockOfEpoch[i])
		if err != nil {
			return nil, err
		}
		err = b.ProcessEpochMetaData(epochMetaData, validatorsStakeAmount, validatorsStateChanged)
		if err != nil {
			return nil, err
		}
		// TODO: Calculate RandomSeed
	}

	// TODO: load accountState of all the dilithium pk in validatorsStateChanged
	// then update the stake balance, pending stake balance and the balance
	// accountState must be loaded based on the trie of parentHeaderHash

	// TODO: Temporary random seed calculation
	var randomSeed int64
	h := md5.New()
	h.Write(parentHeaderHash[:])
	randomSeed = int64(binary.BigEndian.Uint64(h.Sum(nil)))

	currentEpoch := slotNumber / blocksPerEpoch
	epochMetaData.AllotSlots(randomSeed, currentEpoch, parentHeaderHash)

	return epochMetaData, nil
}

func GetBlockStorageKey(blockHeaderHash common.Hash) []byte {
	return []byte(fmt.Sprintf("BLOCK-%s", blockHeaderHash))
}

func GetBlock(db *db.DB, blockHeaderHash common.Hash) (*Block, error) {
	data, err := db.Get(GetBlockStorageKey(blockHeaderHash))
	if err != nil {
		return nil, err
	}

	b := &Block{}
	return b, b.DeSerialize(data)
}

func BlockFromPBData(block *protos.Block) *Block {
	return &Block{HeaderFromPBData(block.Header), block}
}
