package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/zond/block/rewards"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"github.com/theQRL/zond/storagekeys"
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

func (h *Header) PBData() *protos.BlockHeader {
	return h.pbData
}

func (h *Header) Number() *big.Int {
	return big.NewInt(int64(h.pbData.SlotNumber))
}

func (h *Header) Hash() common.Hash {
	return common.BytesToHash(h.pbData.Hash)
}

func (h *Header) ParentHash() common.Hash {
	return common.BytesToHash(h.pbData.ParentHash)
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

func (h *Header) Timestamp() *big.Int {
	return big.NewInt(int64(h.pbData.TimestampSeconds))
}

func (h *Header) Root() common.Hash {
	return common.BytesToHash(h.pbData.Root)
}

func (h *Header) TransactionsRoot() common.Hash {
	return common.BytesToHash(h.pbData.TransactionsRoot)
}

func (h *Header) ReceiptsRoot() common.Hash {
	return common.BytesToHash(h.pbData.ReceiptsRoot)
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
	address := misc.GetDilithiumAddressFromUnSizedPK(b.pbData.ProtocolTransactions[0].GetPk())
	return &address
}

func (b *Block) Hash() common.Hash {
	return b.Header().Hash()
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

func (b *Block) partialBlockHashableBytes(tmp *bytes.Buffer) {
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
}

func (b *Block) PartialBlockSigningHash() common.Hash {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction
	tmp := new(bytes.Buffer)
	b.partialBlockHashableBytes(tmp)

	h := sha256.New()
	h.Write(tmp.Bytes())

	return common.BytesToHash(h.Sum(nil))
}

func (b *Block) BlockSigningHash() common.Hash {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction

	tmp := new(bytes.Buffer)
	b.partialBlockHashableBytes(tmp)

	for i := 1; i < len(b.ProtocolTransactions()); i++ {
		tx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[i])
		txHash := tx.TxHash(tx.GetSigningHash(b.PartialBlockSigningHash()))
		tmp.Write(txHash[:])
	}

	h := sha256.New()
	h.Write(tmp.Bytes())

	return common.BytesToHash(h.Sum(nil))
}

func (b *Block) Attest(chainID uint64, d *dilithium.Dilithium, attestorNonce uint64) (*transactions.Attest, error) {
	attestTx := transactions.NewAttest(chainID, attestorNonce)
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
	blockSigningHash := b.BlockSigningHash()
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	message := coinBaseTx.GetSigningHash(blockSigningHash)
	coinBaseTx.Sign(d, message[:])
	b.ProtocolTransactions()[0] = coinBaseTx.PBData()

	blockHash := ComputeBlockHash(b)
	// Set hash in block header
	b.pbData.Header.Hash = blockHash[:]
}

func ComputeBlockHash(b *Block) common.Hash {
	blockSigningHash := b.BlockSigningHash()
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	message := coinBaseTx.GetSigningHash(blockSigningHash)

	// Compute block hash
	tmp := new(bytes.Buffer)
	tmp.Write(blockSigningHash[:])
	txHash := coinBaseTx.TxHash(message)
	tmp.Write(txHash[:])

	headerHash := sha256.New()
	headerHash.Write(tmp.Bytes())
	return common.BytesToHash(headerHash.Sum(nil))
}

func NewBlock(chainId uint64, timestamp uint64, proposerDilithiumPK []byte, slotNumber uint64,
	parentHeaderHash common.Hash, txs []*protos.Transaction, protocolTxs []*protos.ProtocolTransaction,
	signerNonce uint64) *Block {
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
		// TODO: freeReward needs to be calculated based on Gas remaining after refunding gas
		feeReward += tx.Gas * tx.GasPrice
	}

	blockReward := rewards.GetBlockReward()
	attestorReward := rewards.GetAttestorReward()
	coinBase := transactions.NewCoinBase(chainId, proposerDilithiumPK, blockReward,
		attestorReward, feeReward, signerNonce)

	b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, coinBase.PBData())

	for _, tx := range protocolTxs {
		b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, tx)
	}

	return b
}

func (b *Block) UpdateBloom(protocolTxBloom, txBloom [256]byte) {
	b.pbData.Header.ProtocolTxBloom = protocolTxBloom[:]
	b.pbData.Header.TxBloom = txBloom[:]
}

func (b *Block) UpdateRoots(trieRoot common.Hash) {
	b.pbData.Header.Root = trieRoot[:]
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

	bm, err = metadata.GetBlockMetaData(db, bm.ParentHeaderHash())
	if err != nil {
		log.Error("[UpdateFinalizedEpoch] Failed to GetBlockMetaData")
		return err
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
			log.Error("[UpdateFinalizedEpoch] Failed To Load GetBlockMetaData for ", misc.BytesToHexStr(headerHash[:]))
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

func (b *Block) GetPendingValidatorsUpdate() [][]byte {
	pendingStakeValidatorsUpdateMap := make(map[string]uint8)
	var pendingStakeValidatorsUpdate [][]byte
	for _, pbData := range b.Transactions() {
		switch pbData.Type.(type) {
		case *protos.Transaction_Stake:
			strPK := misc.BytesToHexStr(pbData.GetPk())
			_, ok := pendingStakeValidatorsUpdateMap[strPK]
			if !ok {
				pendingStakeValidatorsUpdateMap[strPK] = 0
				pendingStakeValidatorsUpdate = append(pendingStakeValidatorsUpdate, pbData.GetPk())
			}
		}
	}
	return pendingStakeValidatorsUpdate
}

func (b *Block) GetBlockProposer() common.Address {
	tx := b.ProtocolTransactions()[0]
	return misc.GetAddressFromUnSizedPK(tx.GetPk())
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

func GetBlockByNumber(db *db.DB, slotNumber uint64) (*Block, error) {
	blockHeaderHash, err := db.Get(storagekeys.GetBlockHashStorageKeyBySlotNumber(slotNumber))
	if err != nil {
		return nil, err
	}

	return GetBlock(db, common.BytesToHash(blockHeaderHash))
}

func BlockFromPBData(block *protos.Block) *Block {
	return &Block{HeaderFromPBData(block.Header), block}
}
