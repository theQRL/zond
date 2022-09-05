package view

import (
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainBlockHeader struct {
	HeaderHash string `json:"headerHash" bson:"headerHash"`
	SlotNumber uint64 `json:"slotNumber" bson:"slotNumber"`
	Timestamp  uint64 `json:"timestamp" bson:"timestamp"`
	ParentHash string `json:"parentHash" bson:"parentHash"`
}

func (bh *PlainBlockHeader) BlockHeaderFromPBData(bh2 *protos.BlockHeader,
	headerHash common.Hash) {
	bh.HeaderHash = misc.BytesToHexStr(headerHash[:])
	bh.SlotNumber = bh2.SlotNumber
	bh.Timestamp = bh2.TimestampSeconds
	bh.ParentHash = misc.BytesToHexStr(bh2.ParentHash)
}

type PlainBlock struct {
	Header *PlainBlockHeader `json:"header" bson:"header"`
	//Transactions []transactions.PlainTransactionInterface `json:"transactions" bson:"transactions"`
}

func (b *PlainBlock) BlockFromPBData(b2 *protos.Block, headerHash common.Hash) {
	b.Header = &PlainBlockHeader{}
	b.Header.BlockHeaderFromPBData(b2.Header, headerHash)
	//for _, tx := range b2.Transactions {
	//	b.Transactions = append(b.Transactions, transactions.ProtoToPlainTransaction(tx))
	//}
}

func NewPlainBlockHeader(bh2 *protos.BlockHeader, headerHash common.Hash) *PlainBlockHeader {
	return &PlainBlockHeader{
		HeaderHash: misc.BytesToHexStr(headerHash[:]),
		SlotNumber: bh2.SlotNumber,
		Timestamp:  bh2.TimestampSeconds,
		ParentHash: misc.BytesToHexStr(bh2.ParentHash),
	}
}
