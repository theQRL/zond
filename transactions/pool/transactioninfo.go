package pool

import (
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/transactions"
)

type TransactionInfo struct {
	tx          transactions.TransactionInterface
	txHash      common.Hash
	blockNumber uint64
	timestamp   uint64
	index       int
	priority    uint64
}

func (t *TransactionInfo) Transaction() transactions.TransactionInterface {
	return t.tx
}

func (t *TransactionInfo) BlockNumber() uint64 {
	return t.blockNumber
}

func (t *TransactionInfo) Timestamp() uint64 {
	return t.timestamp
}

func (t *TransactionInfo) TxHash() common.Hash {
	return t.txHash
}

func (t *TransactionInfo) CheckOTSExist(tx2 transactions.TransactionInterface) bool {
	return t.tx.OTSIndex() == tx2.OTSIndex()
}

func (t *TransactionInfo) IsStale(currentBlockHeight uint64) bool {
	if currentBlockHeight > t.blockNumber+config.GetUserConfig().TransactionPool.StaleTransactionThreshold {
		return true
	}

	// If chain recovered from a fork where chain height is reduced
	// then update block_number of the transactions in pool
	if currentBlockHeight < t.blockNumber {
		t.UpdateBlockNumber(currentBlockHeight)
	}

	return false
}

func (t *TransactionInfo) UpdateBlockNumber(currentBlockHeight uint64) {
	t.blockNumber = currentBlockHeight
}

func CreateTransactionInfo(tx transactions.TransactionInterface, txHash common.Hash,
	blockNumber uint64, timestamp uint64) *TransactionInfo {
	t := &TransactionInfo{}
	t.tx = tx
	t.txHash = txHash
	t.blockNumber = blockNumber
	t.timestamp = timestamp
	t.priority = 0 // TODO: Replace priority with tx type and fee

	return t
}
