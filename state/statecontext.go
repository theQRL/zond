package state

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/address"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"go.etcd.io/bbolt"
	"reflect"
)

type StateContext struct {
	db             *db.DB
	addressesState map[string]*address.AddressState
	dilithiumState map[string]*metadata.DilithiumMetaData
	slaveState     map[string]*metadata.SlaveMetaData
	otsIndexState  map[string]*metadata.OTSIndexMetaData


	slotNumber uint64
	blockProposer []byte
	parentBlockHeaderHash []byte
	blockHeaderHash []byte
	partialBlockSigningHash []byte
	blockSigningHash []byte

	validatorsToXMSSAddress map[string][]byte

	epochMetaData *metadata.EpochMetaData
	epochBlockHashes *metadata.EpochBlockHashes
	mainChainMetaData *metadata.MainChainMetaData
}

func (s *StateContext) GetSlotNumber() uint64 {
	return s.slotNumber
}

func (s *StateContext) PartialBlockSigningHash() []byte {
	return s.partialBlockSigningHash
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

func (s *StateContext) PrepareAddressState(addr string) error {
	strKey := misc.Bin2HStr(address.GetAddressStateKey(misc.HStr2Bin(addr)))
	_, ok := s.addressesState[strKey]
	if ok {
		return nil
	}

	addressState, err := address.GetAddressState(s.db, misc.HStr2Bin(addr),
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	s.addressesState[strKey] = addressState

	return err
}

func (s *StateContext) GetAddressState(addr string) (*address.AddressState, error) {
	strKey := misc.Bin2HStr(address.GetAddressStateKey(misc.HStr2Bin(addr)))
	addressState, ok := s.addressesState[strKey]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Address %s not found in addressesState", addr))
	}
	return addressState, nil
}

func (s *StateContext) GetAddressStateByPK(pk []byte) (*address.AddressState, error) {
	addr := misc.Bin2HStr(misc.PK2BinAddress(pk))
	return s.GetAddressState(addr)
}

func (s *StateContext) PrepareDilithiumMetaData(dilithiumPK string) error {
	strKey := misc.Bin2HStr(metadata.GetDilithiumMetaDataKey(misc.HStr2Bin(dilithiumPK)))
	_, ok := s.dilithiumState[strKey]
	if ok {
		return nil
	}

	dilithiumMetaData, err := metadata.GetDilithiumMetaData(s.db, misc.HStr2Bin(dilithiumPK),
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	s.dilithiumState[strKey] = dilithiumMetaData
	return err
}

func (s *StateContext) AddDilithiumMetaData(dilithiumPK string, dilithiumMetaData *metadata.DilithiumMetaData) error {
	strKey := misc.Bin2HStr(metadata.GetDilithiumMetaDataKey(misc.HStr2Bin(dilithiumPK)))
	_, ok := s.dilithiumState[strKey]
	if ok {
		return errors.New("DilithiumPK already exists")
	}
	s.dilithiumState[strKey] = dilithiumMetaData
	return nil
}

func (s *StateContext) GetDilithiumState(dilithiumPK string) *metadata.DilithiumMetaData {
	strKey := misc.Bin2HStr(metadata.GetDilithiumMetaDataKey(misc.HStr2Bin(dilithiumPK)))
	dilithiumState, _ := s.dilithiumState[strKey]
	return dilithiumState
}

func (s *StateContext) PrepareSlaveMetaData(masterAddr string, slavePK string) error {
	strKey := misc.Bin2HStr(metadata.GetSlaveMetaDataKey(misc.HStr2Bin(masterAddr), misc.HStr2Bin(slavePK)))
	_, ok := s.slaveState[strKey]
	if ok {
		return nil
	}

	slaveMetaData, err := metadata.GetSlaveMetaData(s.db, misc.HStr2Bin(masterAddr), misc.HStr2Bin(slavePK),
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	s.slaveState[strKey] = slaveMetaData
	return err
}

func (s *StateContext) AddSlaveMetaData(masterAddr string, slavePK string,
	slaveMetaData *metadata.SlaveMetaData) error {
	strKey := misc.Bin2HStr(metadata.GetSlaveMetaDataKey(misc.HStr2Bin(masterAddr), misc.HStr2Bin(slavePK)))
	_, ok := s.slaveState[strKey]
	if ok {
		return errors.New("SlaveMetaData already exists")
	}
	s.slaveState[strKey] = slaveMetaData
	return nil
}

func (s *StateContext) GetSlaveState(masterAddr string, slavePK string) *metadata.SlaveMetaData {
	strKey := misc.Bin2HStr(metadata.GetSlaveMetaDataKey(misc.HStr2Bin(masterAddr), misc.HStr2Bin(slavePK)))
	slaveMetaData, _ := s.slaveState[strKey]
	return slaveMetaData
}

func (s *StateContext) PrepareOTSIndexMetaData(address string, otsIndex uint64) error {
	key := metadata.GetOTSIndexMetaDataKeyByOTSIndex(misc.HStr2Bin(address), otsIndex)
	strKey := misc.Bin2HStr(key)
	_, ok := s.otsIndexState[strKey]
	if ok {
		return nil
	}

	otsIndexMetaData, err := metadata.GetOTSIndexMetaData(s.db, misc.HStr2Bin(address), otsIndex,
		s.parentBlockHeaderHash, s.mainChainMetaData.FinalizedBlockHeaderHash())
	s.otsIndexState[strKey] = otsIndexMetaData
	return err
}

func (s *StateContext) AddOTSIndexMetaData(address string, otsIndex uint64,
	otsIndexMetaData *metadata.OTSIndexMetaData) error {
	strKey := misc.Bin2HStr(metadata.GetOTSIndexMetaDataKeyByOTSIndex(misc.HStr2Bin(address), otsIndex))
	_, ok := s.otsIndexState[strKey]
	if ok {
		return errors.New("OTSIndexMetaData already exists")
	}
	s.otsIndexState[strKey] = otsIndexMetaData
	return nil
}

func (s *StateContext) GetOTSIndexState(address string, otsIndex uint64) *metadata.OTSIndexMetaData {
	strKey := misc.Bin2HStr(metadata.GetOTSIndexMetaDataKeyByOTSIndex(misc.HStr2Bin(address), otsIndex))
	otsIndexMetaData, _ := s.otsIndexState[strKey]
	return otsIndexMetaData
}

func (s *StateContext) Commit(blockStorageKey []byte, bytesBlock []byte, isFinalizedState bool) error {
	blockMetaData := metadata.NewBlockMetaData(s.parentBlockHeaderHash, s.blockHeaderHash, s.slotNumber)
	var parentBlockMetaData *metadata.BlockMetaData
	var err error
	if s.slotNumber != 0 {
		parentBlockMetaData, err = metadata.GetBlockMetaData(s.db, s.parentBlockHeaderHash)
		if  err != nil {
			log.Error("Failed to load Parent BlockMetaData")
			return err
		}
		parentBlockMetaData.AddChildHeaderHash(s.blockHeaderHash)
	}
	return s.db.DB().Update(func(tx *bbolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("DB"))
		if err := blockMetaData.Commit(b); err != nil {
			log.Error("Failed to commit BlockMetaData")
			return err
		}
		err = s.epochBlockHashes.AddHeaderHashBySlotNumber(s.blockHeaderHash, s.slotNumber)
		if err != nil {
			log.Error("Failed to Add HeaderHash into EpochBlockHashes")
			return err
		}
		if err:= s.epochBlockHashes.Commit(b); err != nil {
			log.Error("Failed to commit EpochBlockHashes")
			return err
		}

		if s.slotNumber != 0 {
			if err := parentBlockMetaData.Commit(b); err != nil {
				log.Error("Failed to commit ParentBlockMetaData")
				return err
			}
		}

		if s.slotNumber == 0 || blockMetaData.Epoch() != parentBlockMetaData.Epoch() {
			if err := s.epochMetaData.Commit(b); err != nil {
				log.Error("Failed to commit EpochMetaData")
				return err
			}
		}

		err = b.Put(blockStorageKey, bytesBlock)
		if err != nil {
			log.Error("Failed to commit block")
			return err
		}

		if isFinalizedState {
			// Update Main Chain Finalized Block Data
			s.mainChainMetaData.UpdateFinalizedBlockData(s.blockHeaderHash, s.slotNumber)
			s.mainChainMetaData.UpdateLastBlockData(s.blockHeaderHash, s.slotNumber)
			if err := s.mainChainMetaData.Commit(b); err != nil {
				log.Error("Failed to commit MainChainMetaData")
				return err
			}
		}
		if reflect.DeepEqual(s.parentBlockHeaderHash, s.mainChainMetaData.LastBlockHeaderHash()) {
			// Update Main Chain Last Block Data
			s.mainChainMetaData.UpdateLastBlockData(s.blockHeaderHash, s.slotNumber)
			if err := s.mainChainMetaData.Commit(b); err != nil {
				log.Error("Failed to commit MainChainMetaData")
				return err
			}
		}

		if !isFinalizedState {
			b, err = tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(s.blockHeaderHash))
			if err != nil {
				log.Error("Failed to create bucket")
				return err
			}
		}
		for _, addressState := range s.addressesState {
			if err := addressState.Commit(b); err != nil {
				log.Error("Failed to commit AddressState")
				return err
			}
		}
		for _, dilithiumMetaData := range s.dilithiumState {
			if err := dilithiumMetaData.Commit(b); err != nil {
				log.Error("Failed to commit DilithiumMetaData")
				return err
			}
		}

		for _, slaveMetaData := range s.slaveState {
			if err := slaveMetaData.Commit(b); err != nil {
				log.Error("Failed to commit SlaveMetaData")
				return err
			}
		}

		for _, otsIndexMetaData := range s.otsIndexState {
			if err := otsIndexMetaData.Commit(b); err != nil {
				log.Error("Failed to commit OtsIndexMetaData")
				return err
			}
		}

		return nil
	})
}

func NewStateContext(db *db.DB, slotNumber uint64, blockProposer []byte,
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

	return &StateContext {
		db:             db,
		addressesState: make(map[string]*address.AddressState),
		dilithiumState: make(map[string]*metadata.DilithiumMetaData),
		slaveState:     make(map[string]*metadata.SlaveMetaData),
		otsIndexState:  make(map[string]*metadata.OTSIndexMetaData),

		slotNumber: slotNumber,
		blockProposer: blockProposer,
		parentBlockHeaderHash: parentBlockHeaderHash,
		blockHeaderHash: blockHeaderHash,
		partialBlockSigningHash: partialBlockSigningHash,
		blockSigningHash: blockSigningHash,
		validatorsToXMSSAddress: make(map[string][]byte),

		epochMetaData: epochMetaData,
		epochBlockHashes: epochBlockHashes,
		mainChainMetaData: mainChainMetaData,
	}, nil
}
