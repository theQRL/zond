package pool

import (
	"errors"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/transactions"
	"reflect"
)

type PriorityQueue []*TransactionInfo

func (pq PriorityQueue) Full() bool {
	return uint64(pq.Len()) > config.GetConfig().User.TransactionPool.TransactionPoolSize
}

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(newTi *TransactionInfo) error {
	isValidXMSS := xmss.IsValidXMSSAddress(newTi.tx.AddrFrom())
	if pq != nil {
		for _, ti := range *pq {
			if reflect.DeepEqual(ti.TxHash(), newTi.TxHash()) {
				return errors.New("transaction already exists in pool")
			}

			if isValidXMSS && reflect.DeepEqual(ti.tx.PK(), newTi.tx.PK()) {
				if ti.CheckOTSExist(newTi.tx) {
					return errors.New("a transaction already exists signed with same ots key")
				}
			}
		}
	}

	n := len(*pq)
	transactionInfo := newTi
	transactionInfo.index = n
	*pq = append(*pq, transactionInfo)
	return nil
}

func (pq *PriorityQueue) Pop() *TransactionInfo {
	old := *pq
	n := len(old)
	if n == 0 {
		return nil
	}
	transactionInfo := old[n-1]
	transactionInfo.index = -1 // for safety
	*pq = old[:n-1]
	return transactionInfo
}

func (pq *PriorityQueue) Remove(tx transactions.TransactionInterface, txHash common.Hash) bool {
	if pq != nil {
		for index, ti := range *pq {
			if reflect.DeepEqual(ti.TxHash(), txHash) {
				pq.removeByIndex(index)
				return true
			}
		}
	}
	return false
}

func (pq *PriorityQueue) removeByIndex(index int) {
	*pq = append((*pq)[:index], (*pq)[index+1:]...)
}

func (pq *PriorityQueue) RemoveTxInBlock(block *block.Block) {
	if pq == nil {
		return
	}
	for _, protoTX := range block.Transactions() {
		tx := transactions.ProtoToTransaction(protoTX)
		pq.Remove(tx, tx.Hash())
	}
}

func (pq *PriorityQueue) Contains(newTI *TransactionInfo) bool {
	if pq != nil {
		for _, ti := range *pq {
			if reflect.DeepEqual(ti.TxHash(), newTI.TxHash()) {
				return true
			}
		}
	}
	return false
}
