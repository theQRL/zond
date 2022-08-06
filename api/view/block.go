package view

import (
	"encoding/hex"
	"github.com/theQRL/zond/common"
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
	bh.HeaderHash = hex.EncodeToString(headerHash[:])
	bh.SlotNumber = bh2.SlotNumber
	bh.Timestamp = bh2.TimestampSeconds
	bh.ParentHash = hex.EncodeToString(bh2.ParentHash)
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
