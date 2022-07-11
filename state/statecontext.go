package state

import (
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/address"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"go.etcd.io/bbolt"
	"math/big"
	"reflect"
)

type StateContext struct {
	db             *db.DB
	addressesState map[string]*address.AddressState
	dilithiumState map[string]*metadata.DilithiumMetaData
	slaveState     map[string]*metadata.SlaveMetaData
	otsIndexState  map[string]*metadata.OTSIndexMetaData

	slotNumber                   uint64
	blockProposer                []byte
	finalizedHeaderHash          []byte
	parentBlockHeaderHash        []byte
	blockHeaderHash              []byte
	partialBlockSigningHash      []byte
	blockSigningHash             []byte
	currentBlockTotalStakeAmount uint64

	validatorsToXMSSAddress map[string][]byte
	attestorsFlag           map[string]bool // Flag just to mark if attestor has been processed, need not to be stored in state
	blockProposerFlag       bool            // Flag just to mark once block propose has been processed, need not to be stored in state

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

func (s *StateContext) PartialBlockSigningHash() []byte {
	return s.partialBlockSigningHash
}

func (s *StateContext) SetPartialBlockSigningHash(p []byte) {
	s.partialBlockSigningHash = p
}

func (s *StateContext) BlockSigningHash() []byte {
	return s.blockSigningHash
}

func (s *StateContext) BlockProposer() []byte {
	return s.blockProposer
}

func (s *StateContext) ValidatorsToXMSSAddress() map[string][]byte {
	return s.validatorsToXMSSAddress
}

func (s *StateContext) processValidatorStakeAmount(dilithiumPK []byte) error {
	strKey := hex.EncodeToString(metadata.GetDilithiumMetaDataKey(dilithiumPK))
	slotLeaderDilithiumMetaData, ok := s.dilithiumState[strKey]
	if !ok {
		return errors.New(fmt.Sprintf("validator dilithium state not found for %s", dilithiumPK))
	}
	s.currentBlockTotalStakeAmount += slotLeaderDilithiumMetaData.Balance()
	return nil
}

func (s *StateContext) ProcessAttestorsFlag(attestorDilithiumPK []byte) error {
	if s.slotNumber == 0 {
		return nil
	}
	strAttestorDilithiumPK := hex.EncodeToString(attestorDilithiumPK)
	result, ok := s.attestorsFlag[strAttestorDilithiumPK]
	if !ok {
		return errors.New("attestor is not assigned to attest at this slot number")
	}

	if result {
		return errors.New("attestor already attested for this slot number")
	}

	err := s.processValidatorStakeAmount(attestorDilithiumPK)
	if err != nil {
		return err
	}
	s.attestorsFlag[strAttestorDilithiumPK] = true
	return nil
}

func (s *StateContext) ProcessBlockProposerFlag(blockProposerDilithiumPK []byte) error {
	if s.slotNumber == 0 {
		return nil
	}
	slotInfo := s.epochMetaData.SlotInfo()[s.slotNumber%config.GetDevConfig().BlocksPerEpoch]
	slotLeader := s.epochMetaData.Validators()[slotInfo.SlotLeader]
	if !reflect.DeepEqual(slotLeader, blockProposerDilithiumPK) {
		return errors.New("unexpected block proposer")
	}
	if s.blockProposerFlag {
		return errors.New("block proposer has already been processed")
	}

	err := s.processValidatorStakeAmount(blockProposerDilithiumPK)
	if err != nil {
		return err
	}
	s.blockProposerFlag = true
	return nil
}

func (s *StateContext) PrepareAddressState(addr string) error {
	binAddr, err := hex.DecodeString(addr)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(address.GetAddressStateKey(binAddr))
	_, ok := s.addressesState[strKey]
	if ok {
		return nil
	}

	addressState, err := address.GetAddressState(s.db, binAddr,
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	if addressState == nil {
		return err
	}
	s.addressesState[strKey] = addressState

	return err
}

func (s *StateContext) GetAddressState(addr string) (*address.AddressState, error) {
	binAddr, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	strKey := hex.EncodeToString(address.GetAddressStateKey(binAddr))
	addressState, ok := s.addressesState[strKey]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Address %s not found in addressesState", addr))
	}
	return addressState, nil
}

func (s *StateContext) GetAddressStateByPK(pk []byte) (*address.AddressState, error) {
	address := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(pk))
	addr := hex.EncodeToString(address[:])
	return s.GetAddressState(addr)
}

func (s *StateContext) PrepareValidatorsToXMSSAddress(dilithiumPK []byte) error {
	xmssAddress, err := metadata.GetXMSSAddressFromDilithiumPK(s.db,
		dilithiumPK, s.parentBlockHeaderHash, s.finalizedHeaderHash)
	if err != nil {
		log.Error("Failed to PrepareValidatorsToXMSSAddress for ",
			hex.EncodeToString(dilithiumPK))
		return err
	}
	s.validatorsToXMSSAddress[hex.EncodeToString(dilithiumPK)] = xmssAddress
	return nil
}

func (s *StateContext) GetXMSSAddressByDilithiumPK(dilithiumPK []byte) []byte {
	return s.validatorsToXMSSAddress[hex.EncodeToString(dilithiumPK)]
}

func (s *StateContext) PrepareDilithiumMetaData(dilithiumPK string) error {
	binDilithiumPK, err := hex.DecodeString(dilithiumPK)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(metadata.GetDilithiumMetaDataKey(binDilithiumPK))
	_, ok := s.dilithiumState[strKey]
	if ok {
		return nil
	}

	dilithiumMetaData, err := metadata.GetDilithiumMetaData(s.db, binDilithiumPK,
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	if err != nil {
		return err
	}
	s.dilithiumState[strKey] = dilithiumMetaData
	return err
}

func (s *StateContext) AddDilithiumMetaData(dilithiumPK string, dilithiumMetaData *metadata.DilithiumMetaData) error {
	binDilithiumPK, err := hex.DecodeString(dilithiumPK)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(metadata.GetDilithiumMetaDataKey(binDilithiumPK))
	_, ok := s.dilithiumState[strKey]
	if ok {
		return errors.New("dilithiumPK already exists")
	}
	s.dilithiumState[strKey] = dilithiumMetaData
	return nil
}

func (s *StateContext) GetDilithiumState(dilithiumPK string) *metadata.DilithiumMetaData {
	binDilithiumPk, err := hex.DecodeString(dilithiumPK)
	if err != nil {
		log.Error("Error decoding dilithium PK")
		return nil
	}
	strKey := hex.EncodeToString(metadata.GetDilithiumMetaDataKey(binDilithiumPk))
	dilithiumState, _ := s.dilithiumState[strKey]
	return dilithiumState
}

func (s *StateContext) PrepareSlaveMetaData(masterAddr string, slavePK string) error {
	binMasterAddress, err := hex.DecodeString(masterAddr)
	if err != nil {
		return err
	}
	binSlavePk, err := hex.DecodeString(slavePK)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(metadata.GetSlaveMetaDataKey(binMasterAddress, binSlavePk))
	_, ok := s.slaveState[strKey]
	if ok {
		return nil
	}

	slaveMetaData, err := metadata.GetSlaveMetaData(s.db, binMasterAddress, binSlavePk,
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	if slaveMetaData == nil {
		return err
	}
	s.slaveState[strKey] = slaveMetaData
	return err
}

func (s *StateContext) AddSlaveMetaData(masterAddr string, slavePK string,
	slaveMetaData *metadata.SlaveMetaData) error {
	binMasterAddress, err := hex.DecodeString(masterAddr)
	if err != nil {
		return err
	}
	binSlavePk, err := hex.DecodeString(slavePK)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(metadata.GetSlaveMetaDataKey(binMasterAddress, binSlavePk))
	_, ok := s.slaveState[strKey]
	if ok {
		return errors.New("SlaveMetaData already exists")
	}
	s.slaveState[strKey] = slaveMetaData
	return nil
}

func (s *StateContext) GetSlaveState(masterAddr string, slavePK string) *metadata.SlaveMetaData {
	binMasterAddress, err := hex.DecodeString(masterAddr)
	if err != nil {
		log.Error("Error decoding masterAddr ", err.Error())
		return nil
	}
	binSlavePk, err := hex.DecodeString(slavePK)
	if err != nil {
		log.Error("Error decoding slavePK ", err.Error())
		return nil
	}
	strKey := hex.EncodeToString(metadata.GetSlaveMetaDataKey(binMasterAddress, binSlavePk))
	slaveMetaData, _ := s.slaveState[strKey]
	return slaveMetaData
}

func (s *StateContext) PrepareOTSIndexMetaData(address string, otsIndex uint64) error {
	binAddress, err := hex.DecodeString(address)
	if err != nil {
		return err
	}
	key := metadata.GetOTSIndexMetaDataKeyByOTSIndex(binAddress, otsIndex)
	strKey := hex.EncodeToString(key)
	_, ok := s.otsIndexState[strKey]
	if ok {
		return nil
	}

	otsIndexMetaData, err := metadata.GetOTSIndexMetaData(s.db, binAddress, otsIndex,
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	if otsIndexMetaData == nil {
		return err
	}
	s.otsIndexState[strKey] = otsIndexMetaData
	return err
}

func (s *StateContext) AddOTSIndexMetaData(address string, otsIndex uint64,
	otsIndexMetaData *metadata.OTSIndexMetaData) error {
	binAddress, err := hex.DecodeString(address)
	if err != nil {
		return err
	}
	strKey := hex.EncodeToString(metadata.GetOTSIndexMetaDataKeyByOTSIndex(binAddress, otsIndex))
	_, ok := s.otsIndexState[strKey]
	if ok {
		return errors.New("OTSIndexMetaData already exists")
	}
	s.otsIndexState[strKey] = otsIndexMetaData
	return nil
}

func (s *StateContext) GetOTSIndexState(address string, otsIndex uint64) *metadata.OTSIndexMetaData {
	binAddress, err := hex.DecodeString(address)
	if err != nil {
		log.Error("Error decoding address ", err.Error())
		return nil
	}
	strKey := hex.EncodeToString(metadata.GetOTSIndexMetaDataKeyByOTSIndex(binAddress, otsIndex))
	otsIndexMetaData, _ := s.otsIndexState[strKey]
	return otsIndexMetaData
}

func (s *StateContext) Commit(blockStorageKey []byte, bytesBlock []byte, isFinalizedState bool) error {
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
		if err != nil {
			log.Error("[Commit] Failed to load last block meta data ",
				hex.EncodeToString(s.mainChainMetaData.LastBlockHeaderHash()))
			return err
		}
		err = lastBlockTotalStakeAmount.UnmarshalText(lastBlockMetaData.TotalStakeAmount())
		if err != nil {
			log.Error("[Commit] Unable to Unmarshal Text for lastblockmetadata total stake amount ",
				hex.EncodeToString(s.mainChainMetaData.LastBlockHeaderHash()))
			return err
		}
	}

	currentBlockStakeAmount := big.NewInt(0)
	currentBlockStakeAmount.SetUint64(s.currentBlockTotalStakeAmount)
	totalStakeAmount.Add(totalStakeAmount, currentBlockStakeAmount)
	bytesTotalStakeAmount, err := totalStakeAmount.MarshalText()
	if err != nil {
		log.Error("[Commit] Unable to marshal total stake amount")
		return err
	}

	blockMetaData := metadata.NewBlockMetaData(s.parentBlockHeaderHash, s.blockHeaderHash,
		s.slotNumber, bytesTotalStakeAmount)
	return s.db.DB().Update(func(tx *bbolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("DB"))
		if err := blockMetaData.Commit(b); err != nil {
			log.Error("[Commit] Failed to commit BlockMetaData")
			return err
		}
		err = s.epochBlockHashes.AddHeaderHashBySlotNumber(s.blockHeaderHash, s.slotNumber)
		if err != nil {
			log.Error("[Commit] Failed to Add HeaderHash into EpochBlockHashes")
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
		}

		if !isFinalizedState {
			b, err = tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(s.blockHeaderHash))
			if err != nil {
				log.Error("[Commit] Failed to create bucket")
				return err
			}
		}
		for _, addressState := range s.addressesState {
			if err := addressState.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit AddressState")
				return err
			}
		}
		for _, dilithiumMetaData := range s.dilithiumState {
			if err := dilithiumMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit DilithiumMetaData")
				return err
			}
		}

		for _, slaveMetaData := range s.slaveState {
			if err := slaveMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit SlaveMetaData")
				return err
			}
		}

		for _, otsIndexMetaData := range s.otsIndexState {
			if err := otsIndexMetaData.Commit(b); err != nil {
				log.Error("[Commit] Failed to commit OtsIndexMetaData")
				return err
			}
		}

		return nil
	})
}

func (s *StateContext) Finalize(blockMetaDataPathForFinalization []*metadata.BlockMetaData) error {
	bm := blockMetaDataPathForFinalization[len(blockMetaDataPathForFinalization)-1]
	parentBlockMetaData, err := metadata.GetBlockMetaData(s.db, bm.ParentHeaderHash())
	if err != nil {
		log.Error("[Finalize] Failed to load ParentBlockMetaData ",
			hex.EncodeToString(bm.ParentHeaderHash()))
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

			if !reflect.DeepEqual(parentBlockMetaData.HeaderHash(), bm.ParentHeaderHash()) {
				log.Error("[Finalize] Unexpected error parent block header hash not matching")
				log.Error("Expected ParentBlockHeaderHash ",
					hex.EncodeToString(bm.ParentHeaderHash()))
				log.Error("ParentBlockHeaderHash found ",
					hex.EncodeToString(parentBlockMetaData.HeaderHash()))
				return errors.New("unexpected error parent block header hash not matching")
			}

			parentBlockMetaData.UpdateFinalizedChildHeaderHash(bm.HeaderHash())
			err := parentBlockMetaData.Commit(mainBucket)
			if err != nil {
				log.Error("[Finalize] Failed to Commit ParentBlockMetaData ",
					hex.EncodeToString(parentBlockMetaData.HeaderHash()))
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
	blockProposer []byte, finalizedHeaderHash []byte,
	parentBlockHeaderHash []byte, blockHeaderHash []byte,
	partialBlockSigningHash []byte, blockSigningHash []byte,
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
		db:             db,
		addressesState: make(map[string]*address.AddressState),
		dilithiumState: make(map[string]*metadata.DilithiumMetaData),
		slaveState:     make(map[string]*metadata.SlaveMetaData),
		otsIndexState:  make(map[string]*metadata.OTSIndexMetaData),

		slotNumber:              slotNumber,
		blockProposer:           blockProposer,
		finalizedHeaderHash:     finalizedHeaderHash,
		parentBlockHeaderHash:   parentBlockHeaderHash,
		blockHeaderHash:         blockHeaderHash,
		partialBlockSigningHash: partialBlockSigningHash,
		blockSigningHash:        blockSigningHash,
		validatorsToXMSSAddress: make(map[string][]byte),
		attestorsFlag:           attestorsFlag,
		blockProposerFlag:       false,

		epochMetaData:     epochMetaData,
		epochBlockHashes:  epochBlockHashes,
		mainChainMetaData: mainChainMetaData,
	}, nil
}
