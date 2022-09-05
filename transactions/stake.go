package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
)

type Stake struct {
	Transaction
}

func (tx *Stake) Amount() uint64 {
	return tx.pbData.GetStake().Amount
}

func (tx *Stake) GetSigningHash() common.Hash {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.ChainID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	binary.Write(tmp, binary.BigEndian, tx.Gas())
	binary.Write(tmp, binary.BigEndian, tx.GasPrice())

	binary.Write(tmp, binary.BigEndian, tx.Amount())

	h := sha256.New()
	h.Write(tmp.Bytes())

	output := h.Sum(nil)
	return common.BytesToHash(output)
}

func (tx *Stake) GenerateTxHash() common.Hash {
	return tx.generateTxHash(tx.GetSigningHash())
}

func (tx *Stake) validateData(stateContext *state.StateContext) bool {
	return true
}

func (tx *Stake) Validate(stateContext *state.StateContext) bool {
	return true
}

func (tx *Stake) ApplyStateChanges(stateContext *state.StateContext) error {

	return nil
}

/*
	We are only taking DilithiumPK for the staking purpose, however in the
	future, we may allow other signature schemes like XMSS for staking
*/
func NewStake(chainID uint64, amount uint64,
	gas uint64, gasPrice uint64, nonce uint64, dilithiumPK []byte) *Stake {
	tx := &Stake{}

	tx.pbData = &protos.Transaction{}
	tx.pbData.ChainId = chainID
	tx.pbData.Type = &protos.Transaction_Stake{Stake: &protos.Stake{}}

	tx.pbData.Pk = dilithiumPK
	tx.pbData.Gas = gas
	tx.pbData.GasPrice = gasPrice
	tx.pbData.Nonce = nonce
	tx.pbData.GetStake().Amount = amount

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}

func StakeTransactionFromPBData(pbData *protos.Transaction) *Stake {
	switch pbData.Type.(type) {
	case *protos.Transaction_Stake:
		return &Stake{
			Transaction{
				pbData: pbData,
			},
		}
	default:
		panic("pbData is not a stake transaction")
	}
}
