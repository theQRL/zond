package block

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/chain/rewards"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"reflect"
)

type Block struct {
	pbData *protos.Block
}

func (b *Block) Header() *protos.BlockHeader {
	return b.pbData.Header
}

func (b *Block) Timestamp() uint64 {
	return b.pbData.Header.TimestampSeconds
}

func (b *Block) ParentHeaderHash() []byte {
	return b.pbData.Header.ParentHeaderHash
}

func (b *Block) Epoch() uint64 {
	return b.pbData.Header.SlotNumber / config.GetDevConfig().BlocksPerEpoch
}

func (b *Block) SlotNumber() uint64 {
	return b.pbData.Header.SlotNumber
}

func (b *Block) HeaderHash() []byte {
	blockSigningHash := b.BlockSigningHash()
	tmp := new(bytes.Buffer)
	tmp.Write(blockSigningHash)
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	tmp.Write(coinBaseTx.TxHash(coinBaseTx.GetSigningHash(blockSigningHash)))

	headerHash := misc.NewUCharVector()
	headerHash.AddBytes(tmp.Bytes())
	headerHash.New(goqrllib.Sha2_256(headerHash.GetData()))

	return headerHash.GetBytes()
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
	return proto.Unmarshal(data, b.pbData)
}

func (b *Block) PartialBlockSigningHash() []byte {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction

	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, b.Header().TimestampSeconds)
	binary.Write(tmp, binary.BigEndian, b.Header().SlotNumber)
	tmp.Write(b.Header().ParentHeaderHash)

	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		tmp.Write(tx.TxHash(tx.GetSigningHash()))
	}
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	tmp.Write(coinBaseTx.GetUnsignedHash())

	partialSigningHash := misc.NewUCharVector()
	partialSigningHash.AddBytes(tmp.Bytes())
	partialSigningHash.New(goqrllib.Sha2_256(partialSigningHash.GetData()))

	return partialSigningHash.GetBytes()
}

func (b *Block) BlockSigningHash() []byte {
	// Partial Block Signing Hash is calculated by appending
	// all block info including transaction hashes.
	// It doesn't include coinbase & attestor transaction

	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, b.Header().TimestampSeconds)
	binary.Write(tmp, binary.BigEndian, b.Header().SlotNumber)
	tmp.Write(b.Header().ParentHeaderHash)

	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		tmp.Write(tx.TxHash(tx.GetSigningHash()))
	}
	coinBaseTx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[0])
	tmp.Write(coinBaseTx.GetUnsignedHash())
	for i := 1; i < len(b.ProtocolTransactions()); i++ {
		tx := transactions.ProtoToProtocolTransaction(b.ProtocolTransactions()[i])
		tmp.Write(tx.TxHash(tx.GetSigningHash(b.PartialBlockSigningHash())))
	}


	signingHash := misc.NewUCharVector()
	signingHash.AddBytes(tmp.Bytes())
	signingHash.New(goqrllib.Sha2_256(signingHash.GetData()))

	return signingHash.GetBytes()
}

func (b *Block) Attest(networkID uint64, d *dilithium.Dilithium) (*transactions.Attest, error) {
	attestTx := transactions.NewAttest(networkID, b.ProtocolTransactions()[0].Nonce)
	attestTx.Sign(d, attestTx.GetSigningHash(b.PartialBlockSigningHash()))
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
	coinbaseTx.Sign(d, message)
	b.ProtocolTransactions()[0] = coinbaseTx.PBData()
}

func (b *Block) ProcessEpochMetaData(epochMetaData *metadata.EpochMetaData,
	validatorsStakeAmount map[string]uint64) error {
	for _, pbData := range b.Transactions() {
		switch pbData.Type.(type) {
		case *protos.Transaction_Stake:
			tx := transactions.ProtoToTransaction(pbData)
			err := tx.ApplyEpochMetaData(epochMetaData)
			if err != nil {
				return err
			}
		}
	}
	for _, pbData := range b.ProtocolTransactions() {
		strPK := misc.Bin2HStr(pbData.Pk)
		amount, ok := validatorsStakeAmount[strPK]
		if !ok {
			return errors.New(fmt.Sprintf("balance not loaded for the validator %s", strPK))
		}
		epochMetaData.AddTotalStakeAmountFound(amount)
	}
	return nil
}

func (b *Block) CommitGenesis(db *db.DB, blockProposerXMSSAddress []byte) error {
	blockHeader := b.Header()
	blockHeaderHash := b.HeaderHash()

	blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()

	epochMetaData := metadata.NewEpochMetaData(0, b.ParentHeaderHash(), make([][]byte, 0))

	stateContext, err := state.NewStateContext(db, blockHeader.SlotNumber, blockProposerDilithiumPK,
		blockHeader.ParentHeaderHash, blockHeaderHash, b.PartialBlockSigningHash(),
		b.BlockSigningHash(), epochMetaData)
	if err != nil {
		return err
	}

	if err := stateContext.PrepareAddressState(misc.Bin2HStr(blockProposerXMSSAddress)); err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// Loading States for related address, Dilithium PK, slaves etc.
	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.SetAffectedAddress(stateContext); err != nil {
			return err
		}
	}

	// Validating Transactions
	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		if !tx.Validate(stateContext) {
			return errors.New(fmt.Sprintf("Transaction Validation failed %s",
				tx.TxHash(tx.Signature())))
		}
	}

	validatorsStakeAmount := make(map[string]uint64)

	// Applying State Changes by Transactions
	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.ApplyStateChanges(stateContext); err != nil {
			return err
		}
		switch pbData.Type.(type) {
		case *protos.Transaction_Stake:
			for _, dilithiumPK := range pbData.GetStake().DilithiumPks {
				strDilithiumPK := misc.Bin2HStr(dilithiumPK)
				dilithiumMetaData := stateContext.GetDilithiumState(strDilithiumPK)
				validatorsStakeAmount[strDilithiumPK] = dilithiumMetaData.Balance()
			}
		}
	}

	// Validating Protocol Transactions
	for _, pbData := range b.ProtocolTransactions() {
		tx := transactions.ProtoToProtocolTransaction(pbData)
		if err := tx.SetAffectedAddress(stateContext); err != nil {
			return err
		}
	}

	// Validating Protocol Transactions
	for _, pbData := range b.ProtocolTransactions() {
		tx := transactions.ProtoToProtocolTransaction(pbData)
		if !tx.Validate(stateContext) {
			return errors.New(fmt.Sprintf("Protocol Transaction Validation failed %s",
				tx.TxHash(tx.Signature())))
		}
	}

	// Applying Protocol State Changes by Transactions
	for _, pbData := range b.ProtocolTransactions() {
		tx := transactions.ProtoToProtocolTransaction(pbData)
		if err := tx.ApplyStateChanges(stateContext); err != nil {
			return err
		}
	}

	err = b.ProcessEpochMetaData(epochMetaData, validatorsStakeAmount)
	if err != nil {
		log.Error("Failed to Process Epoch MetaData")
		return err
	}

	// For Genesis Block total stake found and alloted must be same
	epochMetaData.UpdatePrevEpochStakeData(epochMetaData.TotalStakeAmountFound(),
		epochMetaData.TotalStakeAmountFound())

	var randomSeed int64
	h := md5.New()
	h.Write(b.ParentHeaderHash())
	randomSeed = int64(binary.BigEndian.Uint64(h.Sum(nil)))

	currentEpoch := uint64(0)
	epochMetaData.AllotSlots(randomSeed, currentEpoch, b.ParentHeaderHash())

	/* TODO:
	1. Add code to check new finality, if finality reached move it to finalized state
	2. Ensure Number of validators should never be less than 2x of blocks per epoch
	3. De-stake must not be allowed if number of validators reduces to less than 2x of blocks per epoch
	*/
	bytesBlock, err := b.Serialize()
	if err != nil {
		return err
	}
	return stateContext.Commit(GetBlockStorageKey(blockHeaderHash), bytesBlock, true)
}

func (b *Block) Commit(db *db.DB, finalizedHeaderHash []byte, isFinalizedState bool) error {
	/* TODO:
	1. Calculate EpochMetaData
	2. Validate Block
	3. Validate Transaction in the Block
	4. Process Block
	*/

	blockHeader := b.Header()
	blockHeaderHash := b.HeaderHash()
	parentHeaderHash := b.ParentHeaderHash()
	if b.SlotNumber() == 0 {
		parentHeaderHash = b.HeaderHash()
	}
	parentBlock, err := GetBlock(db, parentHeaderHash)
	if err != nil {
		log.Error("[Commit] Error getting Parent Block ", misc.Bin2HStr(parentHeaderHash))
		return err
	}
	if b.Timestamp() <= parentBlock.Timestamp() {
		log.Error("[Commit] Block Timestamp must be greater than Parent Block Timestamp")
		log.Error("Parent Block Timestamp ", parentBlock.Timestamp())
		log.Error("Block Timestamp ", b.Timestamp())
		return err
	}

	parentBlockMetaData, err := metadata.GetBlockMetaData(db, blockHeader.ParentHeaderHash)
	if err != nil {
		log.Error("[Commit] Failed to get Parent Block MetaData")
		return err
	}
	epochMetaData, err := CalculateEpochMetaData(db, blockHeader.SlotNumber,
		blockHeader.ParentHeaderHash, parentBlockMetaData.SlotNumber())
	if err != nil {
		log.Error("Failed to Calculate Epoch MetaData")
		return err
	}

	blockProposerDilithiumPK := b.ProtocolTransactions()[0].GetPk()
	stateContext, err := state.NewStateContext(db, blockHeader.SlotNumber, blockProposerDilithiumPK,
		blockHeader.ParentHeaderHash, blockHeaderHash, b.PartialBlockSigningHash(),
		b.BlockSigningHash(), epochMetaData)

	if err != nil {
		return err
	}

	validatorsToXMSSAddress := stateContext.ValidatorsToXMSSAddress()
	for i := 1; i < len(b.ProtocolTransactions()); i++ {
		pbData := b.ProtocolTransactions()[i]
		// Get XMSS Address based on Dilithium PK
		xmssAddress, err := metadata.GetXMSSAddressFromDilithiumPK(db, pbData.GetPk(),
			parentHeaderHash, finalizedHeaderHash)
		if err != nil {
			return err
		}
		validatorsToXMSSAddress[misc.Bin2HStr(pbData.GetPk())] = xmssAddress
		if err := stateContext.PrepareAddressState(misc.Bin2HStr(xmssAddress)); err != nil {
			return err
		}
	}

	// Get XMSS Address based on Dilithium PK
	blockProposerXMSSAddress, err := metadata.GetXMSSAddressFromDilithiumPK(db,
		blockProposerDilithiumPK, parentHeaderHash, finalizedHeaderHash)
	validatorsToXMSSAddress[misc.Bin2HStr(blockProposerDilithiumPK)] = blockProposerXMSSAddress
	if err := stateContext.PrepareAddressState(misc.Bin2HStr(blockProposerXMSSAddress)); err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// Loading States for related address, Dilithium PK, slaves etc.
	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.SetAffectedAddress(stateContext); err != nil {
			return err
		}
	}

	for _, pbData := range b.ProtocolTransactions() {
		tx := transactions.ProtoToProtocolTransaction(pbData)
		if err := tx.SetAffectedAddress(stateContext); err != nil {
			return err
		}
	}

	// Validate & Apply State Changes by Transactions
	for _, pbData := range b.Transactions() {
		tx := transactions.ProtoToTransaction(pbData)
		if !tx.Validate(stateContext) {
			return errors.New(fmt.Sprintf("Transaction Validation failed %s",
				tx.TxHash(tx.Signature())))
		}
		if err := tx.ApplyStateChanges(stateContext); err != nil {
			return err
		}
	}

	// Validate & Apply Protocol State Changes by Protocol Transactions
	for _, pbData := range b.ProtocolTransactions() {
		tx := transactions.ProtoToProtocolTransaction(pbData)
		if !tx.Validate(stateContext) {
			return errors.New(fmt.Sprintf("Protocol Transaction Validation failed %s",
				tx.TxHash(tx.Signature())))
		}
		if err := tx.ApplyStateChanges(stateContext); err != nil {
			return err
		}
	}

	/* TODO:
	1. Add code to check new finality, if finality reached move it to finalized state
	2. Ensure Number of validators should never be less than 2x of blocks per epoch
	3. De-stake must not be allowed if number of validators reduces to less than 2x of blocks per epoch
	*/

	/*
	1. Load MainChainState
	2. Check if finalized block epoch and current block epoch difference is >= 2
	3. If condition 2 is true, then check if finality achieved
	4. Check must only be done on the first block of the epoch
	 */

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	parentEpoch := parentBlockMetaData.SlotNumber() / blocksPerEpoch
	epoch := b.SlotNumber() / blocksPerEpoch

	if parentEpoch != epoch {
		b.UpdateFinalizedEpoch(db, stateContext)
	}


	bytesBlock, err := b.Serialize()
	if err != nil {
		return err
	}
	return stateContext.Commit(GetBlockStorageKey(blockHeaderHash), bytesBlock, isFinalizedState)
}

func (b *Block) Revert() bool {

	return true
}

func NewBlock(networkId uint64, timestamp uint64, proposerAddress []byte, slotNumber uint64,
	parentHeaderHash []byte, txs []*protos.Transaction, protocolTxs []*protos.ProtocolTransaction,
	lastCoinBaseNonce uint64) *Block {
	b := &Block {
		pbData: &protos.Block{},
	}

	blockHeader := &protos.BlockHeader{}
	blockHeader.TimestampSeconds = timestamp
	blockHeader.SlotNumber = slotNumber
	blockHeader.ParentHeaderHash = parentHeaderHash

	b.pbData.Header = blockHeader

	feeReward := uint64(0)
	for _, tx := range txs {
		b.pbData.Transactions = append(b.pbData.Transactions, tx)
		feeReward += tx.Fee
	}

	blockReward := rewards.GetBlockReward()
	coinBase := transactions.NewCoinBase(networkId, proposerAddress, blockReward,
		blockReward, feeReward, lastCoinBaseNonce)

	b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, coinBase.PBData())

	for _, tx := range protocolTxs {
		b.pbData.ProtocolTransactions = append(b.pbData.ProtocolTransactions, tx)
	}

	return b
}

func (b *Block) UpdateFinalizedEpoch(db *db.DB, stateContext *state.StateContext) error {
	currentEpochMetaData := stateContext.GetEpochMetaData()
	// Ignore Finalization if TotalStakeAmountFound is less than the 2/3rd of TotalStakeAmountAlloted
	if currentEpochMetaData.TotalStakeAmountFound() * 3 < currentEpochMetaData.TotalStakeAmountAlloted() * 2 {
		return nil
	}

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	currentEpoch := b.Epoch()
	finalizedBlockEpoch := stateContext.GetMainChainMetaData().FinalizedBlockSlotNumber() / blocksPerEpoch

	if currentEpoch - finalizedBlockEpoch <= 2 {
		return nil
	}

	bm, err := metadata.GetBlockMetaData(db, b.ParentHeaderHash())
	if err != nil {
		log.Error("[UpdateFinalizedEpoch] Failed to GetBlockMetaData")
		return err
	}

	for ;; {
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

	epochMetaData, err := metadata.GetEpochMetaData(db, bm.SlotNumber(), bm.ParentHeaderHash())
	if err != nil {
		log.Error("[UpdateFinalizedEpoch] Failed to load EpochMetaData for ", bm.Epoch() - 1)
		return err
	}
	if epochMetaData.TotalStakeAmountFound() * 3 < epochMetaData.TotalStakeAmountAlloted() * 2 {
		return nil
	}

	headerHash := bm.ParentHeaderHash()
	blockMetaDataPathForFinalization := make([]*metadata.BlockMetaData, 0)
	for ;; {
		bm, err := metadata.GetBlockMetaData(db, headerHash)
		if err != nil {
			log.Error("[UpdateFinalizedEpoch] Failed To Load GetBlockMetaData for ", misc.Bin2HStr(headerHash))
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
	parentHeaderHash []byte, parentSlotNumber uint64) (*metadata.EpochMetaData, error) {

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
	var pathToFirstBlockOfEpoch [][]byte
	if parentBlockMetaData.SlotNumber() == 0 {
		pathToFirstBlockOfEpoch = append(pathToFirstBlockOfEpoch, parentBlockMetaData.HeaderHash())
	} else {
		for ;epoch == parentEpoch; {
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

	firstBlockOfEpochHeaderHash := pathToFirstBlockOfEpoch[lenPathToFirstBlockOfEpoch - 1]
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
	totalStakeAmountAlloted := uint64(0)
	mainChainMetaData, err := metadata.GetMainChainMetaData(db)
	if err != nil {
		log.Error("[CalculateEpochMetaData] Failed to get MainChainMetaData")
		return nil, err
	}
	for _, dilithiumPK := range epochMetaData.Validators() {
		// TODO: Get Dilithium Public Key Balance
		dilithiumMetaData, err := metadata.GetDilithiumMetaData(db, dilithiumPK,
		pathToFirstBlockOfEpoch[0], mainChainMetaData.FinalizedBlockHeaderHash())
		if err != nil {
			log.Error("[CalculateEpochMetaData] Failed to get DilithiumMetaData")
			return nil, err
		}
		validatorsStakeAmount[misc.Bin2HStr(dilithiumPK)] = dilithiumMetaData.Balance()
	}
	epochMetaData.UpdatePrevEpochStakeData(0,
		totalStakeAmountAlloted)
	for i := lenPathToFirstBlockOfEpoch - 1; i >= 0; i-- {
		b, err := GetBlock(db, pathToFirstBlockOfEpoch[i])
		if err != nil {
			return nil, err
		}
		err = b.ProcessEpochMetaData(epochMetaData, validatorsStakeAmount)
		if err != nil {
			return nil, err
		}
		// TODO: Calculate RandomSeed
	}

	// TODO: Temporary random seed calculation
	var randomSeed int64
	h := md5.New()
	h.Write(parentHeaderHash)
	randomSeed = int64(binary.BigEndian.Uint64(h.Sum(nil)))

	currentEpoch := slotNumber / blocksPerEpoch
	epochMetaData.AllotSlots(randomSeed, currentEpoch, parentHeaderHash)

	return epochMetaData, nil
}

func GetBlockStorageKey(blockHeaderHash []byte) []byte {
	return []byte(fmt.Sprintf("BLOCK-%s", blockHeaderHash))
}

func GetBlock(db *db.DB, blockHeaderHash []byte) (*Block, error) {
	data, err  := db.Get(GetBlockStorageKey(blockHeaderHash))
	if err != nil {
		return nil, err
	}

	b := &Block{
		pbData: &protos.Block{},
	}
	return b, b.DeSerialize(data)
}

func BlockFromPBData(pbData *protos.Block) *Block {
	return &Block { pbData }
}
