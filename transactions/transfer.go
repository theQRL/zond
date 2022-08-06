package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/common/math"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"math/big"
)

type Transfer struct {
	Transaction
}

func (tx *Transfer) Type() uint8 {
	return 0
}

func (tx *Transfer) To() *common.Address {
	to := misc.UnSizedAddressToSizedAddress(tx.pbData.GetTransfer().To)
	return (*common.Address)(&to)
}

func (tx *Transfer) Value() uint64 {
	return tx.pbData.GetTransfer().Value
}

func (tx *Transfer) Data() []byte {
	return tx.pbData.GetTransfer().Data
}

func (tx *Transfer) GetSigningHash() []byte {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.NetworkID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	binary.Write(tmp, binary.BigEndian, tx.Gas())
	binary.Write(tmp, binary.BigEndian, tx.GasPrice())

	to := tx.To()
	tmp.Write(to[:])
	binary.Write(tmp, binary.BigEndian, tx.Value())

	tmp.Write(tx.Data())

	// TODO: Add access list

	h := sha256.New()
	h.Write(tmp.Bytes())

	return h.Sum(nil)
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

func NewTransfer(networkID uint64, to []byte, value uint64, gas uint64, gasPrice uint64,
	data []byte, nonce uint64, pk []byte) *Transfer {
	tx := &Transfer{}

	tx.pbData = &protos.Transaction{}
	tx.pbData.NetworkId = networkID
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

func (tx *Transfer) AsMessage(baseFee *big.Int) (types.Message, error) {
	bigIntGasPrice := big.NewInt(int64(tx.GasPrice()))
	bigIntGasFeeCap := big.NewInt(int64(tx.GasFeeCap()))
	bigIntGasTipCap := big.NewInt(int64(tx.GasTipCap()))
	bigIntValue := big.NewInt(int64(tx.Value()))
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		bigIntGasPrice = math.BigMin(bigIntGasPrice.Add(bigIntGasTipCap, baseFee), bigIntGasFeeCap)
	}

	msg := types.NewMessage(tx.AddrFrom(), tx.To(), tx.Nonce(), bigIntValue, tx.Gas(), bigIntGasPrice, bigIntGasFeeCap, bigIntGasTipCap, tx.Data(), nil, false)

	return msg, nil
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
