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

type TransactionMetaData struct {
	pbData *protos.TransactionMetaData
}

func (t *TransactionMetaData) BlockHash() common.Hash {
	return common.BytesToHash(t.pbData.BlockHash)
}

func (t *TransactionMetaData) BlockNumber() uint64 {
	return t.pbData.BlockNumber
}

func (t *TransactionMetaData) Index() uint64 {
	return t.pbData.Index
}

func (t *TransactionMetaData) Transaction() *protos.Transaction {
	return t.pbData.Transaction
}

func (t *TransactionMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(t.pbData)
}

func (t *TransactionMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, t.pbData)
}

func (t *TransactionMetaData) Commit(b *bbolt.Bucket) error {
	data, err := t.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetTransactionMetaDataKey(common.BytesToHash(t.pbData.Transaction.Hash)), data)
}

func NewTransactionMetaData(blockHash common.Hash, blockNumber uint64,
	index uint64, tx *protos.Transaction) *TransactionMetaData {
	pbData := &protos.TransactionMetaData{
		BlockHash:   blockHash[:],
		BlockNumber: blockNumber,
		Index:       index,
		Transaction: tx,
	}
	return &TransactionMetaData{
		pbData: pbData,
	}
}

func GetTransactionMetaData(d *db.DB, txHash common.Hash) (*TransactionMetaData, error) {
	key := GetTransactionMetaDataKey(txHash)
	data, err := d.Get(key)
	if err != nil {
		log.Error("Error loading TransactionMetaData for key ", string(key), err)
		return nil, err
	}
	t := &TransactionMetaData{
		pbData: &protos.TransactionMetaData{},
	}
	return t, t.DeSerialize(data)
}

func GetTransactionMetaDataKey(txHash common.Hash) []byte {
	return []byte(fmt.Sprintf("TX-META-DATA-%s", misc.BytesToHexStr(txHash[:])))
}
