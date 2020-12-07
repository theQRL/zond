package metadata

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
	"reflect"
)

type EpochBlockHashes struct {
	pbData *protos.EpochBlockHashesMetaData
}

func (e *EpochBlockHashes) PBData() *protos.EpochBlockHashesMetaData {
	return e.pbData
}

func (e *EpochBlockHashes) Epoch() uint64 {
	return e.pbData.Epoch
}

func (e *EpochBlockHashes) BlockHashesBySlotNumber() []*protos.BlockHashesBySlotNumber {
	return e.pbData.BlockHashesBySlotNumber
}

func (e *EpochBlockHashes) Serialize() ([]byte, error) {
	return proto.Marshal(e.pbData)
}

func (e *EpochBlockHashes) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, e.pbData)
}

func (e *EpochBlockHashes) AddHeaderHashBySlotNumber(headerHash []byte,
	slotNumber uint64) error {
	c := config.GetDevConfig()
	if slotNumber / c.BlocksPerEpoch != e.Epoch() {
		return errors.New(
			fmt.Sprintf("SlotNumber %d doesn't belong to epoch %d",
				slotNumber, e.Epoch()))
	}
	startSlotNumber := e.Epoch() * c.BlocksPerEpoch
	index := slotNumber - startSlotNumber
	if e.BlockHashesBySlotNumber()[index].SlotNumber != slotNumber {
		return errors.New(
			fmt.Sprintf("Unexpected slot number %d at index %d",
				e.BlockHashesBySlotNumber()[index].SlotNumber,
				index))
	}

	for _, storedHeaderHash := range e.BlockHashesBySlotNumber()[index].HeaderHashes {
		if reflect.DeepEqual(storedHeaderHash, headerHash) {
			return errors.New(
				fmt.Sprintf("Headerhash %s already exists",
					hex.EncodeToString(headerHash)))
		}
	}
	e.BlockHashesBySlotNumber()[index].HeaderHashes = append(
		e.BlockHashesBySlotNumber()[index].HeaderHashes, headerHash)

	return nil
}

func (e *EpochBlockHashes) Commit(b *bbolt.Bucket) error {
	data, err := e.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetEpochBlockHashesKey(e.Epoch()), data)
}

func NewEpochBlockHashes(epoch uint64) *EpochBlockHashes {
	pbData := &protos.EpochBlockHashesMetaData {
		Epoch: epoch,
	}
	startSlotNumber := epoch * config.GetDevConfig().BlocksPerEpoch
	for i := uint64(0); i < config.GetDevConfig().BlocksPerEpoch; i++ {
		blockHashesBySlotNumber := &protos.BlockHashesBySlotNumber {
			SlotNumber: startSlotNumber + i,
		}
		pbData.BlockHashesBySlotNumber = append(
			pbData.BlockHashesBySlotNumber, blockHashesBySlotNumber)
	}
	return &EpochBlockHashes {
		pbData: pbData,
	}
}

func GetEpochBlockHashes(d *db.DB,
	epoch uint64) (*EpochBlockHashes, error) {
	key := GetEpochBlockHashesKey(epoch)
	data, err := d.Get(key)
	if err != nil {
		log.Error("Error loading EpochBlockHashes for key ", string(key), err)
		return nil, err
	}
	e := &EpochBlockHashes{
		pbData: &protos.EpochBlockHashesMetaData{},
	}
	return e, e.DeSerialize(data)
}

func GetEpochBlockHashesKey(epoch uint64) []byte {
	return []byte(fmt.Sprintf("EPOCH-BLOCK-HASHES-META-DATA-%d", epoch))
}
