package metadata

import (
	"encoding/hex"
	"github.com/theQRL/zond/config"
)

/*
SlotValidatorsMetaData metadata is not stored into DB, as we generate this based
on validators data already stored by epoch.go in the DB.
*/
type SlotValidatorsMetaData struct {
	slotNumber     uint64
	slotLeaderPK   []byte
	validatorsType map[string]uint8
}

func (s *SlotValidatorsMetaData) GetValidatorsType() map[string]uint8 {
	return s.validatorsType
}

func (s *SlotValidatorsMetaData) GetValidatorType(dilithiumPK []byte) (uint8, bool) {
	value, found := s.validatorsType[hex.EncodeToString(dilithiumPK)]
	return value, found
}

func (s *SlotValidatorsMetaData) GetSlotLeaderPK() []byte {
	return s.slotLeaderPK
}

func (s *SlotValidatorsMetaData) IsAttestor(dilithiumPK string) bool {
	value, found := s.validatorsType[dilithiumPK]
	return found && value == 0
}

func (s *SlotValidatorsMetaData) IsSlotLeader(dilithiumPK string) bool {
	value, found := s.validatorsType[dilithiumPK]
	return found && value == 1
}

func NewSlotValidatorsMetaData(slotNumber uint64, epochMetaData *EpochMetaData) *SlotValidatorsMetaData {
	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	slotLeaderIndex := epochMetaData.SlotInfo()[slotNumber%blocksPerEpoch].SlotLeader
	attestorsIndex := epochMetaData.SlotInfo()[slotNumber%blocksPerEpoch].Attestors
	validators := epochMetaData.Validators()

	validatorsType := make(map[string]uint8)
	slotLeaderPK := epochMetaData.Validators()[slotLeaderIndex]
	validatorsType[hex.EncodeToString(slotLeaderPK[:])] = 1

	for _, attestorIndex := range attestorsIndex {
		validatorPK := validators[attestorIndex]
		validatorsType[hex.EncodeToString(validatorPK[:])] = 0
	}

	return &SlotValidatorsMetaData{
		slotNumber:     slotNumber,
		slotLeaderPK:   slotLeaderPK,
		validatorsType: validatorsType,
	}
}
