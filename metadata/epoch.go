package metadata

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
	"math"
	"math/rand"
	"reflect"
)

type EpochMetaData struct {
	pbData *protos.EpochMetaData
}

func (e *EpochMetaData) PBData() *protos.EpochMetaData {
	return e.pbData
}

func (e *EpochMetaData) Epoch() uint64 {
	return e.pbData.Epoch
}

func (e *EpochMetaData) PrevSlotLastBlockHeaderHash() common.Hash {
	var hash common.Hash
	copy(hash[:], e.pbData.PrevSlotLastBlockHeaderHash)
	return hash
}

func (e *EpochMetaData) SlotInfo() []*protos.SlotInfo {
	return e.pbData.SlotInfo
}

func (e *EpochMetaData) Validators() [][]byte {
	return e.pbData.Validators
}

func (e *EpochMetaData) TotalStakeAmountFound() uint64 {
	return e.pbData.PrevEpochStakeData.TotalStakeAmountFound
}

func (e *EpochMetaData) TotalStakeAmountAlloted() uint64 {
	return e.pbData.PrevEpochStakeData.TotalStakeAmountAlloted
}

func (e *EpochMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(e.pbData)
}

func (e *EpochMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, e.pbData)
}

func (e *EpochMetaData) Commit(b *bbolt.Bucket) error {
	data, err := e.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetEpochMetaDataKey(e.Epoch(), e.PrevSlotLastBlockHeaderHash()), data)
}

func (e *EpochMetaData) UpdatePrevEpochStakeData(totalStakeAmountFound uint64, totalStakeAmountAlloted uint64) {
	e.pbData.PrevEpochStakeData.TotalStakeAmountFound = totalStakeAmountFound
	e.pbData.PrevEpochStakeData.TotalStakeAmountAlloted = totalStakeAmountAlloted
}

func (e *EpochMetaData) AddTotalStakeAmountFound(amount uint64) {
	e.pbData.PrevEpochStakeData.TotalStakeAmountFound += amount
}

func (e *EpochMetaData) AddValidators(dilithiumPK []byte) {
	found := false
	for _, validator := range e.Validators() {
		if reflect.DeepEqual(dilithiumPK, validator) {
			found = true
			break
		}
	}
	if !found {
		e.pbData.Validators = append(e.pbData.Validators, dilithiumPK)
	}
}

func (e *EpochMetaData) RemoveValidators(dilithiumPK []byte) {
	for i, validator := range e.Validators() {
		if reflect.DeepEqual(dilithiumPK, validator) {
			e.pbData.Validators = append(e.pbData.Validators[:i], e.pbData.Validators[i+1:]...)
			break
		}
	}
}

func (e *EpochMetaData) AllotSlots(randomSeed int64, epoch uint64, prevSlotLastBlockHeaderHash common.Hash) {
	e.pbData.Epoch = epoch
	e.pbData.PrevSlotLastBlockHeaderHash = prevSlotLastBlockHeaderHash[:]

	// TODO (cyyber): math/rand is not secure, needs to be replaced
	rand.Seed(randomSeed)
	rand.Shuffle(len(e.pbData.Validators), func(i, j int) {
		e.pbData.Validators[i], e.pbData.Validators[j] = e.pbData.Validators[j], e.pbData.Validators[i]
	})

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	e.pbData.SlotInfo = make([]*protos.SlotInfo, blocksPerEpoch)

	lenValidators := uint64(len(e.Validators()))
	if lenValidators == 0 {
		panic("[AllotSlots] impossible error length of validators is 0")
	}
	maxValidatorsPerSlot := uint64(math.Ceil(float64(lenValidators) / float64(blocksPerEpoch)))
	for i := uint64(0); i < maxValidatorsPerSlot; i++ {
		offset := i * blocksPerEpoch
		for j := uint64(0); j < blocksPerEpoch && (offset+j < lenValidators); j++ {
			if i == 0 {
				e.pbData.SlotInfo[j] = &protos.SlotInfo{}
				e.pbData.SlotInfo[j].SlotLeader = i + j
			} else {
				e.pbData.SlotInfo[j].Attestors = append(e.pbData.SlotInfo[j].Attestors, offset+j)
			}
		}
	}
}

func NewEpochMetaData(epoch uint64, prevSlotLastBlockHeaderHash common.Hash,
	validators [][]byte) *EpochMetaData {
	pbData := &protos.EpochMetaData{
		Epoch:                       epoch,
		PrevSlotLastBlockHeaderHash: prevSlotLastBlockHeaderHash[:],
		Validators:                  validators,
		PrevEpochStakeData: &protos.EpochStakeData{
			TotalStakeAmountFound:   0,
			TotalStakeAmountAlloted: 0,
		},
	}
	return &EpochMetaData{
		pbData: pbData,
	}
}

func GetEpochMetaData(db *db.DB, currentBlockSlotNumber uint64, parentHeaderHash common.Hash) (*EpochMetaData, error) {
	var prevSlotLastBlockHeaderHash common.Hash

	blocksPerEpoch := config.GetDevConfig().BlocksPerEpoch
	epoch := currentBlockSlotNumber / blocksPerEpoch
	parentEpoch := epoch

	if currentBlockSlotNumber == 0 {
		key := GetEpochMetaDataKey(0, parentHeaderHash)
		data, err := db.Get(key)
		if err != nil {
			log.Error("Failed to load EpochMetaData for genesis block")
			return nil, err
		}
		epochMetaData := NewEpochMetaData(0, common.Hash{}, nil)
		return epochMetaData, epochMetaData.DeSerialize(data)
	} else {
		for parentEpoch == epoch {
			parentBlockMetaData, err := GetBlockMetaData(db, parentHeaderHash)
			if err != nil {
				return nil, err
			}
			parentEpoch = parentBlockMetaData.SlotNumber() / blocksPerEpoch
			prevSlotLastBlockHeaderHash = parentBlockMetaData.HeaderHash()
			parentHeaderHash = parentBlockMetaData.ParentHeaderHash()
			if parentBlockMetaData.SlotNumber() == 0 {
				if currentBlockSlotNumber/blocksPerEpoch == 0 {
					prevSlotLastBlockHeaderHash = parentBlockMetaData.ParentHeaderHash()
				}
				break
			}
		}
	}

	key := GetEpochMetaDataKey(epoch, prevSlotLastBlockHeaderHash)
	data, err := db.Get(key)

	if err != nil {
		log.Error("Error loading EpochMetaData for  ", misc.BytesToHexStr(prevSlotLastBlockHeaderHash[:]),
			err)
		return nil, err
	}

	sm := &EpochMetaData{
		pbData: &protos.EpochMetaData{},
	}
	return sm, sm.DeSerialize(data)
}

func GetEpochMetaDataKey(epoch uint64, prevSlotLastBlockHeaderHash common.Hash) []byte {
	return []byte(fmt.Sprintf("EPOCH-META-DATA-%d-%s", epoch, prevSlotLastBlockHeaderHash))
}
