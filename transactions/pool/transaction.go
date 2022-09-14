package pool

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/transactions"
	"sync"
)

type TransactionPool struct {
	lock   sync.Mutex
	txPool *PriorityQueue
	config *config.Config
	ntp    ntp.NTPInterface
	nonce  map[common.Address]uint64
}

func (t *TransactionPool) isFull() bool {
	return t.txPool.Full()
}

func (t *TransactionPool) IsFull() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.txPool.Full()
}

func (t *TransactionPool) Contains(tx *TransactionInfo) bool {
	return t.txPool.Contains(tx)
}

func (t *TransactionPool) Add(tx transactions.TransactionInterface, txHash common.Hash,
	slotNumber uint64, timestamp uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.isFull() {
		return errors.New("transaction pool is full")
	}

	if timestamp == 0 {
		timestamp = t.ntp.Time()
	}

	ti := CreateTransactionInfo(tx, txHash, slotNumber, timestamp)

	err := t.txPool.Push(ti)
	if err != nil {
		return err
	}

	t.AddNonceData(tx.PK(), tx.Nonce())
	log.Info("Added Transaction ", misc.BytesToHexStr(txHash[:]), " to pool")

	return nil
}

func (t *TransactionPool) Pop() *TransactionInfo {
	t.lock.Lock()
	defer t.lock.Unlock()

	tx := t.txPool.Pop()

	if tx != nil {
		t.DeleteNonceData(tx.tx.PK(), tx.tx.Nonce())
	}

	return tx
}

func (t *TransactionPool) Remove(tx transactions.TransactionInterface, txHash common.Hash) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.txPool.Remove(tx, txHash) {
		return true
	}

	t.DeleteNonceData(tx.PK(), tx.Nonce())
	return false
}

func (t *TransactionPool) RemoveTxInBlock(block *block.Block) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.txPool.RemoveTxInBlock(block)
	for _, protoTX := range block.Transactions() {
		t.DeleteNonceData(protoTX.Pk, protoTX.Nonce)
	}
}

func (t *TransactionPool) AddTxFromBlock(block *block.Block, currentBlockHeight uint64) error {
	for _, protoTX := range block.Transactions() {
		tx := transactions.ProtoToTransaction(protoTX)
		err := t.Add(tx, tx.Hash(), currentBlockHeight, t.ntp.Time())
		if err != nil {
			return err
		}
		t.AddNonceData(tx.PK(), tx.Nonce())
	}
	return nil
}

func (t *TransactionPool) DeleteNonceData(pk []byte, nonce uint64) {
	address := misc.GetAddressFromUnSizedPK(pk)
	currentNonce, ok := t.nonce[address]
	if ok && currentNonce == nonce {
		delete(t.nonce, address)
	}
}

func (t *TransactionPool) AddNonceData(pk []byte, nonce uint64) {
	address := misc.GetAddressFromUnSizedPK(pk)
	currentNonce, ok := t.nonce[address]
	if (!ok) || (ok && nonce > currentNonce) {
		t.nonce[address] = nonce
	}
}

func (t *TransactionPool) GetNonceByAddress(address common.Address) (uint64, bool) {
	nonce, ok := t.nonce[address]
	return nonce, ok
}

// TODO: Check Stale txn and rebroadcast if required
//func (t *TransactionPool) CheckStale(currentBlockHeight uint64, state *state.State) error {
//	t.lock.Lock()
//	defer t.lock.Unlock()
//
//	txPoolLength := len(*t.txPool)
//	for i := 0; i < txPoolLength; i++ {
//		txInfo := (*t.txPool)[i]
//		if txInfo.IsStale(currentBlockHeight) {
//			tx := txInfo.Transaction()
//			addrFromState, err := address.GetAddressState(tx.AddrFrom())
//			if err != nil {
//				log.Error("Error while getting AddressState",
//					"Txhash", txInfo.TxHash(),
//					"Address", misc.Bin2Qaddress(tx.AddrFrom()),
//					"Error", err.Error())
//				return err
//			}
//			addrFromPKState := addrFromState
//			addrFromPK := tx.GetSlave()
//			if addrFromPK != nil {
//				addrFromPKState, err = state.GetAddressState(addrFromPK)
//				if err != nil {
//					log.Error("Error while getting AddressState",
//						"Txhash", tx.Txhash(),
//						"Address", misc.Bin2Qaddress(tx.GetSlave()),
//						"Error", err.Error())
//					return err
//				}
//			}
//			if !tx.ValidateExtended(addrFromState, addrFromPKState) {
//				t.txPool.removeByIndex(i)
//				i -= 1
//				txPoolLength -= 1
//				continue
//			}
//			// TODO: Chan to Re-Broadcast Txn
//			//txInfo.UpdateBlockNumber(currentBlockHeight)
//			//msg := &generated.Message{
//			//	Msg:&generated.LegacyMessage_T{
//			//		Block:b.PBData(),
//			//	},
//			//	MessageType:generated.LegacyMessage_BK,
//			//}
//			//
//			//registerMessage := &messages.RegisterMessage{
//			//	MsgHash:misc.BytesToHexStr(b.Hash()),
//			//	Msg:msg,
//			//}
//			//select {
//			//case t.registerAndBroadcastChan <- nil:
//			//}
//
//		}
//	}
//	return nil
//}

func CreateTransactionPool() *TransactionPool {
	t := &TransactionPool{
		config: config.GetConfig(),
		ntp:    ntp.GetNTP(),
		txPool: &PriorityQueue{},
		nonce:  make(map[common.Address]uint64),
	}
	return t
}
