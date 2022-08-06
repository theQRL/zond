package metadata

import (
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type SlaveMetaData struct {
	pbData *protos.SlaveMetaData
}

func (s *SlaveMetaData) TxHash() []byte {
	return s.pbData.TxHash
}

func (s *SlaveMetaData) Address() []byte {
	return s.pbData.Address
}

func (s *SlaveMetaData) SlavePK() []byte {
	return s.pbData.SlavePk
}

func (s *SlaveMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(s.pbData)
}

func (s *SlaveMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, s.pbData)
}

func (s *SlaveMetaData) Commit(b *bbolt.Bucket) error {
	data, err := s.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetSlaveMetaDataKey(s.Address(), s.SlavePK()), data)
}

func NewSlaveMetaData(txHash []byte, address []byte, slavePK []byte) *SlaveMetaData {
	pbData := &protos.SlaveMetaData{
		TxHash:  txHash,
		Address: address,
		SlavePk: slavePK,
	}
	return &SlaveMetaData{
		pbData: pbData,
	}
}

func GetSlaveMetaData(db *db.DB, address []byte, slavePK []byte,
	headerHash common.Hash, finalizedHeaderHash common.Hash) (*SlaveMetaData, error) {
	key := GetSlaveMetaDataKey(address, slavePK)

	data, err := GetDataByBucket(db, key, headerHash, finalizedHeaderHash)

	if err != nil {
		log.Error("Error loading SlaveMetaData for key ", key, err)
		return nil, err
	}
	sm := &SlaveMetaData{
		pbData: &protos.SlaveMetaData{},
	}
	return sm, sm.DeSerialize(data)
}

func GetSlaveMetaDataKey(address []byte, slavePK []byte) []byte {
	return []byte(fmt.Sprintf("SLAVE-META-DATA-%s-%s", address, hex.EncodeToString(slavePK)))
}
