package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
)

type Transfer struct {
	Transaction
}

func (tx *Transfer) To() *common.Address {
	if len(tx.pbData.GetTransfer().To) == 0 {
		return nil
	}
	to := misc.UnSizedAddressToSizedAddress(tx.pbData.GetTransfer().To)
	return (*common.Address)(&to)
}

func (tx *Transfer) Value() uint64 {
	return tx.pbData.GetTransfer().Value
}

func (tx *Transfer) Data() []byte {
	return tx.pbData.GetTransfer().Data
}

func (tx *Transfer) GetSigningHash() common.Hash {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.ChainID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())
	binary.Write(tmp, binary.BigEndian, tx.Gas())
	binary.Write(tmp, binary.BigEndian, tx.GasPrice())

	to := tx.To()
	if to != nil {
		tmp.Write(to[:])
	}
	binary.Write(tmp, binary.BigEndian, tx.Value())

	tmp.Write(tx.Data())

	// TODO: Add access list

	h := sha256.New()
	h.Write(tmp.Bytes())

	output := h.Sum(nil)
	return common.BytesToHash(output)
}

func (tx *Transfer) GenerateTxHash() common.Hash {
	return tx.generateTxHash(tx.GetSigningHash())
}

func (tx *Transfer) validateData(stateContext *state.StateContext) bool {
	return true
}

func (tx *Transfer) Validate(stateContext *state.StateContext) bool {
	return true
}

func (tx *Transfer) ApplyStateChanges(stateContext *state.StateContext) error {
	return nil
}

func NewTransfer(chainID uint64, to []byte, value uint64, gas uint64, gasPrice uint64,
	data []byte, nonce uint64, pk []byte) *Transfer {
	tx := &Transfer{}

	tx.pbData = &protos.Transaction{}
	tx.pbData.ChainId = chainID
	tx.pbData.Type = &protos.Transaction_Transfer{Transfer: &protos.Transfer{}}

	tx.pbData.Pk = pk
	tx.pbData.Gas = gas
	tx.pbData.GasPrice = gasPrice
	tx.pbData.Nonce = nonce
	transferPBData := tx.pbData.GetTransfer()
	transferPBData.To = to
	transferPBData.Value = value
	transferPBData.Data = data

	// TODO: Add new slave support during the transaction
	//for _, slavePK := range slavesPKs {
	//	transferPBData.SlavePks = append(transferPBData.SlavePks, slavePK)
	//}

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}

func TransferTransactionFromPBData(pbData *protos.Transaction) *Transfer {
	switch pbData.Type.(type) {
	case *protos.Transaction_Transfer:
		return &Transfer{
			Transaction{
				pbData: pbData,
			},
		}
	default:
		panic("pbData is not a transfer transaction")
	}
}
