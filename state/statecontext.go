package state

import (
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
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

func (s *StateContext) GetSlotNumber() uint64 {
	return s.slotNumber
}

func (s *StateContext) GetMainChainMetaData() *metadata.MainChainMetaData {
	return s.mainChainMetaData
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
	//addr := dilithium.GetDilithiumAddressFromPK(misc.UnSizedDilithiumPKToSizedPK(dilithiumPK))
	//stakeBalance := s.db2.GetStakeBalance(addr)

	if stakeBalance.Uint64() == 0 {
		return errors.New(fmt.Sprintf("Invalid stake balance %d for pk %s", stakeBalance, dilithiumPK))
	}
	s.currentBlockTotalStakeAmount = s.currentBlockTotalStakeAmount.Add(s.currentBlockTotalStakeAmount, stakeBalance)
	return nil
}

func (s *StateContext) ProcessAttestorsFlag(attestorDilithiumPK []byte, stakeBalance *big.Int) error {
	if s.slotNumber == 0 {
		return nil
	}
	strAttestorDilithiumPK := hex.EncodeToString(attestorDilithiumPK)
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
	if s.slotNumber == 0 {
		return nil
	}
	slotInfo := s.epochMetaData.SlotInfo()[s.slotNumber%config.GetDevConfig().BlocksPerEpoch]
	slotLeader := s.epochMetaData.Validators()[slotInfo.SlotLeader]
	if !reflect.DeepEqual(slotLeader, blockProposerDilithiumPK) {
		return errors.New("unexpected block proposer")
	}
	if s.validatorsFlag[hex.EncodeToString(blockProposerDilithiumPK)] {
		return errors.New("block proposer has already been processed")
	}

	err := s.processValidatorStakeAmount(blockProposerDilithiumPK, stakeBalance)
	if err != nil {
		return err
	}
	s.validatorsFlag[hex.EncodeToString(blockProposerDilithiumPK)] = true
	return nil
}

func (s *StateContext) PrepareValidators(dilithiumPK []byte) error {
	s.validatorsFlag[hex.EncodeToString(dilithiumPK)] = false
	return nil
}

func (s *StateContext) Commit(blockStorageKey []byte, bytesBlock []byte, trieRoot common.Hash, isFinalizedState bool) error {
	var parentBlockMetaData *metadata.BlockMetaData
	var err error
	totalStakeAmount := big.NewInt(0)
	lastBlockTotalStakeAmount := big.NewInt(0)

	if s.slotNumber != 0 {
		parentBlockMetaData, err = metadata.GetBlockMetaData(s.db, s.parentBlockHeaderHash)
		if err != nil {
			log.Error("[Commit] Failed to load Parent BlockMetaData")
			return err
		}
		parentBlockMetaData.AddChildHeaderHash(s.blockHeaderHash)

		err = totalStakeAmount.UnmarshalText(parentBlockMetaData.TotalStakeAmount())
		if err != nil {
			log.Error("[Commit] Unable to unmarshal total stake amount of parent block metadata")
			return err
		}

		lastBlockMetaData, err := metadata.GetBlockMetaData(s.db, s.mainChainMetaData.LastBlockHeaderHash())
		lastBlockHash := s.mainChainMetaData.LastBlockHeaderHash()
		if err != nil {
			log.Error("[Commit] Failed to load last block meta data ",
				hex.EncodeToString(lastBlockHash[:]))
			return err
		}
		err = lastBlockTotalStakeAmount.UnmarshalText(lastBlockMetaData.TotalStakeAmount())
		if err != nil {
			log.Error("[Commit] Unable to Unmarshal Text for lastblockmetadata total stake amount ",
				hex.EncodeToString(lastBlockHash[:]))
			return err
		}
	}

	totalStakeAmount = totalStakeAmount.Add(totalStakeAmount, s.currentBlockTotalStakeAmount)
	bytesTotalStakeAmount, err := totalStakeAmount.MarshalText()
	if err != nil {
		log.Error("[Commit] Unable to marshal total stake amount")
		return err
	}

	blockMetaData := metadata.NewBlockMetaData(s.parentBlockHeaderHash, s.blockHeaderHash,
		s.slotNumber, bytesTotalStakeAmount, trieRoot)
	return s.db.DB().Update(func(tx *bbolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("DB"))
		if err := blockMetaData.Commit(b); err != nil {
			log.Error("[Commit] Failed to commit BlockMetaData")
			return err
		}
		err = s.epochBlockHashes.AddHeaderHashBySlotNumber(s.blockHeaderHash, s.slotNumber)
		if err != nil {
			log.Error("[Commit] Failed to Add Hash into EpochBlockHashes")
			return err
		}
		if err := s.epochBlockHashes.Commit(b); err != nil {
			log.Error("[Commit] Failed to commit EpochBlockHashes")
			return err
		}

		if s.slotNumber != 0 {
			if err := parentBlockMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit ParentBlockMetaData")
				return err
			}
		}

		if s.slotNumber == 0 || blockMetaData.Epoch() != parentBlockMetaData.Epoch() {
			if err := s.epochMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit EpochMetaData")
				return err
			}
		}

		err = b.Put(blockStorageKey, bytesBlock)
		if err != nil {
			log.Error("[Commit] Failed to commit block")
			return err
		}

		if isFinalizedState {
			// Update Main Chain Finalized Block Data
			s.mainChainMetaData.UpdateFinalizedBlockData(s.blockHeaderHash, s.slotNumber)
			s.mainChainMetaData.UpdateLastBlockData(s.blockHeaderHash, s.slotNumber)
			if err := s.mainChainMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit MainChainMetaData")
				return err
			}
		}

		if totalStakeAmount.Cmp(lastBlockTotalStakeAmount) == 1 {
			// Update Main Chain Last Block Data
			s.mainChainMetaData.UpdateLastBlockData(s.blockHeaderHash, s.slotNumber)
			if err := s.mainChainMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit MainChainMetaData")
				return err
			}

			err = b.Put([]byte("mainchain-head-trie-root"), trieRoot[:])
			if err != nil {
				log.Error("[Commit] Failed to commit state trie root")
				return err
			}
		}

		if !isFinalizedState {
			b, err = tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(s.blockHeaderHash))
			if err != nil {
				log.Error("[Commit] Failed to create bucket")
				return err
			}
		}

		return nil
	})
}

func (s *StateContext) Finalize(blockMetaDataPathForFinalization []*metadata.BlockMetaData) error {
	bm := blockMetaDataPathForFinalization[len(blockMetaDataPathForFinalization)-1]
	parentBlockMetaData, err := metadata.GetBlockMetaData(s.db, bm.ParentHeaderHash())
	pHash := bm.ParentHeaderHash()
	if err != nil {
		log.Error("[Finalize] Failed to load ParentBlockMetaData ",
			hex.EncodeToString(pHash[:]))
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
					hex.EncodeToString(expectedPHash[:]))
				log.Error("ParentBlockHeaderHash found ",
					hex.EncodeToString(foundPHash[:]))
				return errors.New("unexpected error parent block header hash not matching")
			}

			parentBlockMetaData.UpdateFinalizedChildHeaderHash(bm.HeaderHash())
			err := parentBlockMetaData.Commit(mainBucket)
			if err != nil {
				log.Error("[Finalize] Failed to Commit ParentBlockMetaData ",
					hex.EncodeToString(foundPHash[:]))
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

	attestorsFlag := make(map[string]bool)
	if slotNumber > 0 {
		slotInfo := epochMetaData.SlotInfo()[slotNumber%config.GetDevConfig().BlocksPerEpoch]
		for _, attestorsIndex := range slotInfo.Attestors {
			attestorsFlag[hex.EncodeToString(epochMetaData.Validators()[attestorsIndex])] = false
		}
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

		validatorsFlag: make(map[string]bool),

		epochMetaData:     epochMetaData,
		epochBlockHashes:  epochBlockHashes,
		mainChainMetaData: mainChainMetaData,
	}, nil
}
