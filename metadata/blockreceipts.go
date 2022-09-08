package metadata

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type BlockReceipts struct {
	pbData *protos.BlockReceipts
}

func (b *BlockReceipts) Receipts() []byte {
	return b.pbData.Receipts
}

func (b *BlockReceipts) Serialize() ([]byte, error) {
	return proto.Marshal(b.pbData)
}

func (b *BlockReceipts) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, b.pbData)
}

func NewBlockReceipts(receipts []byte) *BlockReceipts {
	return &BlockReceipts{
		pbData: &protos.BlockReceipts{
			Receipts: receipts,
		},
	}
}

func GetBlockReceipts(d *db.DB, headerHash common.Hash, blockNumber uint64, isProtocolTxReceipt bool) (*BlockReceipts, error) {
	key := GetBlockReceiptsKey(headerHash, blockNumber, isProtocolTxReceipt)
	data, err := d.Get(key)
	if err != nil {
		log.Error("Error loading BlockReceipts for key ", string(key), err)
		return nil, err
	}
	b := &BlockReceipts{
		pbData: &protos.BlockReceipts{},
	}
	return b, b.DeSerialize(data)
}

func GetBlockReceiptsKey(headerHash common.Hash, blockNumber uint64, isProtocolTxReceipt bool) []byte {
	if isProtocolTxReceipt {
		return []byte(fmt.Sprintf("BLOCK-PROTOCOL-TX-RECEIPTS-%s-%d", misc.BytesToHexStr(headerHash[:]), blockNumber))
	}
	return []byte(fmt.Sprintf("BLOCK-TX-RECEIPTS-%s-%d", misc.BytesToHexStr(headerHash[:]), blockNumber))
}
