package view

import (
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainBlockHeader struct {
	HeaderHash       string `json:"headerHash" bson:"headerHash"`
	SlotNumber       uint64 `json:"slotNumber" bson:"slotNumber"`
	Timestamp        uint64 `json:"timestamp" bson:"timestamp"`
	ParentHeaderHash string `json:"parentHeaderHash" bson:"parentHeaderHash"`
}

func (bh *PlainBlockHeader) BlockHeaderFromPBData(bh2 *protos.BlockHeader,
	headerHash []byte) {
	bh.HeaderHash = misc.Bin2HStr(headerHash)
	bh.SlotNumber = bh2.SlotNumber
	bh.Timestamp = bh2.TimestampSeconds
	bh.ParentHeaderHash = misc.Bin2HStr(bh2.ParentHeaderHash)
}

type PlainBlock struct {
	Header       *PlainBlockHeader                        `json:"header" bson:"header"`
	//Transactions []transactions.PlainTransactionInterface `json:"transactions" bson:"transactions"`
}

func (b *PlainBlock) BlockFromPBData(b2 *protos.Block, headerHash []byte) {
	b.Header = &PlainBlockHeader{}
	b.Header.BlockHeaderFromPBData(b2.Header, headerHash)
	//for _, tx := range b2.Transactions {
	//	b.Transactions = append(b.Transactions, transactions.ProtoToPlainTransaction(tx))
	//}
}