package metadata

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type MainChainMetaData struct {
	pbData *protos.MainChainMetaData
}

func (m *MainChainMetaData) FinalizedBlockHeaderHash() common.Hash {
	var hash common.Hash
	copy(hash[:], m.pbData.FinalizedBlockHeaderHash)
	return hash
}

func (m *MainChainMetaData) FinalizedBlockSlotNumber() uint64 {
	return m.pbData.FinalizedBlockSlotNumber
}

func (m *MainChainMetaData) LastBlockHeaderHash() common.Hash {
	var hash common.Hash
	copy(hash[:], m.pbData.LastBlockHeaderHash)
	return hash
}

func (m *MainChainMetaData) LastBlockSlotNumber() uint64 {
	return m.pbData.LastBlockSlotNumber
}

func (m *MainChainMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(m.pbData)
}

func (m *MainChainMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, m.pbData)
}

func (m *MainChainMetaData) Commit(b *bbolt.Bucket) error {
	data, err := m.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetMainChainMetaDataKey(), data)
}

func (m *MainChainMetaData) UpdateFinalizedBlockData(finalizedBlockHeaderHash common.Hash,
	finalizedBlockSlotNumber uint64) {
	m.pbData.FinalizedBlockHeaderHash = finalizedBlockHeaderHash[:]
	m.pbData.FinalizedBlockSlotNumber = finalizedBlockSlotNumber
}

func (m *MainChainMetaData) UpdateLastBlockData(lastBlockHeaderHash common.Hash,
	lastBlockSlotNumber uint64) {
	m.pbData.LastBlockHeaderHash = lastBlockHeaderHash[:]
	m.pbData.LastBlockSlotNumber = lastBlockSlotNumber
}

func NewMainChainMetaData(finalizedBlockHeaderHash common.Hash, finalizedBlockSlotNumber uint64,
	lastBlockHeaderHash common.Hash, lastBlockSlotNumber uint64) *MainChainMetaData {
	pbData := &protos.MainChainMetaData{
		FinalizedBlockHeaderHash: finalizedBlockHeaderHash[:],
		FinalizedBlockSlotNumber: finalizedBlockSlotNumber,

		LastBlockHeaderHash: lastBlockHeaderHash[:],
		LastBlockSlotNumber: lastBlockSlotNumber,
	}
	return &MainChainMetaData{
		pbData: pbData,
	}
}

func GetMainChainMetaData(d *db.DB) (*MainChainMetaData, error) {
	key := GetMainChainMetaDataKey()
	data, err := d.Get(key)
	if err != nil {
		return nil, err
	}
	m := &MainChainMetaData{
		pbData: &protos.MainChainMetaData{},
	}
	return m, m.DeSerialize(data)
}

func GetMainChainMetaDataKey() []byte {
	return []byte(fmt.Sprintf("MAIN-CHAIN-META-DATA"))
}
