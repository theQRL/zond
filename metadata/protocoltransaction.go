package metadata

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type ProtocolTransactionMetaData struct {
	pbData *protos.ProtocolTransactionMetaData
}

func (t *ProtocolTransactionMetaData) BlockHash() common.Hash {
	return common.BytesToHash(t.pbData.BlockHash)
}

func (t *ProtocolTransactionMetaData) BlockNumber() uint64 {
	return t.pbData.BlockNumber
}

func (t *ProtocolTransactionMetaData) Index() uint64 {
	return t.pbData.Index
}

func (t *ProtocolTransactionMetaData) Transaction() *protos.ProtocolTransaction {
	return t.pbData.ProtocolTransaction
}

func (t *ProtocolTransactionMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(t.pbData)
}

func (t *ProtocolTransactionMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, t.pbData)
}

func (t *ProtocolTransactionMetaData) Commit(b *bbolt.Bucket) error {
	data, err := t.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetTransactionMetaDataKey(common.BytesToHash(t.pbData.ProtocolTransaction.Hash)), data)
}

func NewProtocolTransactionMetaData(blockHash common.Hash, blockNumber uint64,
	index uint64, tx *protos.ProtocolTransaction) *ProtocolTransactionMetaData {
	pbData := &protos.ProtocolTransactionMetaData{
		BlockHash:           blockHash[:],
		BlockNumber:         blockNumber,
		Index:               index,
		ProtocolTransaction: tx,
	}
	return &ProtocolTransactionMetaData{
		pbData: pbData,
	}
}

func GetProtocolTransactionMetaData(d *db.DB, txHash common.Hash) (*ProtocolTransactionMetaData, error) {
	key := GetProtocolTransactionMetaDataKey(txHash)
	data, err := d.Get(key)
	if err != nil {
		log.Error("Error loading ProtocolTransactionMetaData for key ", string(key), err)
		return nil, err
	}
	t := &ProtocolTransactionMetaData{
		pbData: &protos.ProtocolTransactionMetaData{},
	}
	return t, t.DeSerialize(data)
}

func GetProtocolTransactionMetaDataKey(txHash common.Hash) []byte {
	return []byte(fmt.Sprintf("PROTOCOL-TX-META-DATA-%s", misc.BytesToHexStr(txHash[:])))
}
