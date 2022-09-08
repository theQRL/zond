package state

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"go.etcd.io/bbolt"
	"math/big"
	"reflect"
)

type StateContext struct {
	db *db.DB

	slotNumber                   uint64
	blockProposer                []byte
	finalizedHeaderHash          common.Hash
	parentBlockHeaderHash        common.Hash
	blockHeaderHash              common.Hash
	partialBlockSigningHash      common.Hash
	blockSigningHash             common.Hash
	currentBlockTotalStakeAmount *big.Int

	validatorsFlag map[string]bool // Flag just to mark if attestor or block proposer has been processed, need not to be stored in state

	epochMetaData     *metadata.EpochMetaData
	epochBlockHashes  *metadata.EpochBlockHashes
	mainChainMetaData *metadata.MainChainMetaData

	totalTransactionFee uint64
}

func (s *StateContext) GetTotalTransactionFee() uint64 {
	return s.totalTransactionFee
}

func (s *StateContext) AddTransactionFee(fee uint64) {
	s.totalTransactionFee += fee
}

func (s *StateContext) GetEpochMetaData() *metadata.EpochMetaData {
	return s.epochMetaData
}

func (s *StateContext) GetDB() *db.DB {
	return s.db
}

func (s *StateContext) GetSlotNumber() uint64 {
	return s.slotNumber
}

func (s *StateContext) GetParentBlockHeaderHash() common.Hash {
	return s.parentBlockHeaderHash
}

func (s *StateContext) GetBlockHeaderHash() common.Hash {
	return s.blockHeaderHash
}

func (s *StateContext) GetCurrentBlockTotalStakeAmount() *big.Int {
	return s.currentBlockTotalStakeAmount
}

func (s *StateContext) GetMainChainMetaData() *metadata.MainChainMetaData {
	return s.mainChainMetaData
}

func (s *StateContext) GetEpochBlockHashes() *metadata.EpochBlockHashes {
	return s.epochBlockHashes
}

func (s *StateContext) PartialBlockSigningHash() common.Hash {
	return s.partialBlockSigningHash
}

func (s *StateContext) SetPartialBlockSigningHash(p common.Hash) {
	s.partialBlockSigningHash = p
}

func (s *StateContext) BlockSigningHash() common.Hash {
	return s.blockSigningHash
}

func (s *StateContext) BlockProposer() []byte {
	return s.blockProposer
}

func (s *StateContext) ValidatorsFlag() map[string]bool {
	return s.validatorsFlag
}

func (s *StateContext) processValidatorStakeAmount(dilithiumPK []byte, stakeBalance *big.Int) error {
	requiredStakeAmount := big.NewInt(int64(config.GetDevConfig().StakeAmount))
	if stakeBalance.Cmp(requiredStakeAmount) < 0 {
		return errors.New(fmt.Sprintf("Invalid stake balance %d for address %s", stakeBalance, misc.GetAddressFromUnSizedPK(dilithiumPK)))
	}
	s.currentBlockTotalStakeAmount = s.currentBlockTotalStakeAmount.Add(s.currentBlockTotalStakeAmount,
		requiredStakeAmount)
	return nil
}

func (s *StateContext) ProcessAttestorsFlag(attestorDilithiumPK []byte, stakeBalance *big.Int) error {
	// TODO (cyyber): Make this cleaner for genesis block
	if s.slotNumber == 0 {
		return nil
	}
	strAttestorDilithiumPK := misc.BytesToHexStr(attestorDilithiumPK)
	result, ok := s.validatorsFlag[strAttestorDilithiumPK]
	if !ok {
		return errors.New("attestor is not assigned to attest at this slot number")
	}

	if result {
		return errors.New("attestor already attested for this slot number")
	}

	err := s.processValidatorStakeAmount(attestorDilithiumPK, stakeBalance)
	if err != nil {
		return err
	}
	s.validatorsFlag[strAttestorDilithiumPK] = true
	return nil
}

func (s *StateContext) ProcessBlockProposerFlag(blockProposerDilithiumPK []byte, stakeBalance *big.Int) error {
	// TODO (cyyber): Make this cleaner for genesis block
	if s.slotNumber == 0 {
		return nil
	}
	slotInfo := s.epochMetaData.SlotInfo()[s.slotNumber%config.GetDevConfig().BlocksPerEpoch]
	slotLeader := s.epochMetaData.Validators()[slotInfo.SlotLeader]
	if !reflect.DeepEqual(slotLeader, blockProposerDilithiumPK) {
		return errors.New("unexpected block proposer")
	}
	result, ok := s.validatorsFlag[misc.BytesToHexStr(blockProposerDilithiumPK)]
	if !ok {
		return errors.New("block proposer is not assigned to this slot number")
	}

	if result {
		return errors.New("block proposer has already been processed")
	}

	err := s.processValidatorStakeAmount(blockProposerDilithiumPK, stakeBalance)
	if err != nil {
		return err
	}
	s.validatorsFlag[misc.BytesToHexStr(blockProposerDilithiumPK)] = true
	return nil
}

func (s *StateContext) PrepareValidators(dilithiumPK []byte) {
	s.validatorsFlag[misc.BytesToHexStr(dilithiumPK)] = false
}

func (s *StateContext) Finalize(blockMetaDataPathForFinalization []*metadata.BlockMetaData) error {
	bm := blockMetaDataPathForFinalization[len(blockMetaDataPathForFinalization)-1]
	parentBlockMetaData, err := metadata.GetBlockMetaData(s.db, bm.ParentHeaderHash())
	pHash := bm.ParentHeaderHash()
	if err != nil {
		log.Error("[Finalize] Failed to load ParentBlockMetaData ",
			misc.BytesToHexStr(pHash[:]))
		return err
	}

	return s.db.DB().Update(func(tx *bbolt.Tx) error {
		var err error
		mainBucket := tx.Bucket([]byte("DB"))
		for i := len(blockMetaDataPathForFinalization) - 1; i >= 0; i-- {
			bm := blockMetaDataPathForFinalization[i]
			blockBucket := tx.Bucket(metadata.GetBlockBucketName(bm.HeaderHash()))
			c := blockBucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				err = mainBucket.Put(k, v)
				if err != nil {
					log.Error("[Finalize] Finalization failed for key = ", k,
						"value ", v)
					return err
				}
			}

			expectedPHash := bm.ParentHeaderHash()
			foundPHash := parentBlockMetaData.HeaderHash()
			if !reflect.DeepEqual(parentBlockMetaData.HeaderHash(), bm.ParentHeaderHash()) {
				log.Error("[Finalize] Unexpected error parent block header hash not matching")
				log.Error("Expected ParentBlockHeaderHash ",
					misc.BytesToHexStr(expectedPHash[:]))
				log.Error("ParentBlockHeaderHash found ",
					misc.BytesToHexStr(foundPHash[:]))
				return errors.New("unexpected error parent block header hash not matching")
			}

			parentBlockMetaData.UpdateFinalizedChildHeaderHash(bm.HeaderHash())
			err := parentBlockMetaData.Commit(mainBucket)
			if err != nil {
				log.Error("[Finalize] Failed to Commit ParentBlockMetaData ",
					misc.BytesToHexStr(foundPHash[:]))
				return err
			}
			parentBlockMetaData = bm
			log.Info("Finalized Block #", bm.SlotNumber())
		}

		bm = blockMetaDataPathForFinalization[0]
		finalizedBlockHeaderHash := bm.HeaderHash()
		finalizedSlotNumber := bm.SlotNumber()
		s.mainChainMetaData.UpdateFinalizedBlockData(finalizedBlockHeaderHash, finalizedSlotNumber)
		return s.mainChainMetaData.Commit(mainBucket)
	})
}

func NewStateContext(db *db.DB, slotNumber uint64,
	blockProposer []byte, finalizedHeaderHash common.Hash,
	parentBlockHeaderHash common.Hash, blockHeaderHash common.Hash,
	partialBlockSigningHash common.Hash, blockSigningHash common.Hash,
	epochMetaData *metadata.EpochMetaData) (*StateContext, error) {

	mainChainMetaData, err := metadata.GetMainChainMetaData(db)
	if err != nil {
		return nil, err
	}

	epoch := slotNumber / config.GetDevConfig().BlocksPerEpoch
	epochBlockHashes, err := metadata.GetEpochBlockHashes(db, epoch)
	if err != nil {
		epochBlockHashes = metadata.NewEpochBlockHashes(epoch)
	}

	validatorsFlag := make(map[string]bool)
	if slotNumber > 0 {
		slotInfo := epochMetaData.SlotInfo()[slotNumber%config.GetDevConfig().BlocksPerEpoch]
		for _, attestorsIndex := range slotInfo.Attestors {
			validatorsFlag[misc.BytesToHexStr(epochMetaData.Validators()[attestorsIndex])] = false
		}
		validatorsFlag[misc.BytesToHexStr(epochMetaData.Validators()[slotInfo.SlotLeader])] = false
	}

	return &StateContext{
		db: db,

		slotNumber:                   slotNumber,
		blockProposer:                blockProposer,
		finalizedHeaderHash:          finalizedHeaderHash,
		parentBlockHeaderHash:        parentBlockHeaderHash,
		blockHeaderHash:              blockHeaderHash,
		partialBlockSigningHash:      partialBlockSigningHash,
		blockSigningHash:             blockSigningHash,
		currentBlockTotalStakeAmount: big.NewInt(0),

		validatorsFlag: validatorsFlag,

		epochMetaData:     epochMetaData,
		epochBlockHashes:  epochBlockHashes,
		mainChainMetaData: mainChainMetaData,
	}, nil
}
