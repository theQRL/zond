package view

import (
	"errors"

	"github.com/theQRL/zond/common/hexutil"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/transactions"
)

type PlainStakeTransaction struct {
	ChainID         uint64 `json:"chainID"`
	Gas             uint64 `json:"gas"`
	GasPrice        uint64 `json:"gasPrice"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           uint64 `json:"nonce"`
	TransactionHash string `json:"transactionHash"`
	TransactionType string `json:"transactionType"`

	Amount uint64 `json:"amount"`
}

type PlainStakeTransactionRPC struct {
	ChainID   string `json:"chainId"`
	Gas       string `json:"gas"`
	GasPrice  string `json:"gasPrice"`
	PublicKey string `json:"pk"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
	Value     string `json:"value"`
}

func (t *PlainStakeTransactionRPC) TransactionFromPBData(tx *protos.Transaction) {
	t.ChainID = hexutil.EncodeUint64(tx.GetChainId()) //tx.ChainId
	t.Gas = hexutil.EncodeUint64(tx.GetGas())
	t.GasPrice = hexutil.EncodeUint64(tx.GetGasPrice())
	t.PublicKey = misc.BytesToHexStr(tx.GetPk())
	t.Signature = misc.BytesToHexStr(tx.GetSignature())
	t.Nonce = hexutil.EncodeUint64(tx.GetNonce())
	t.Value = hexutil.EncodeUint64(tx.GetStake().GetAmount())
}

func (t *PlainStakeTransaction) TransactionFromPBData(tx *protos.Transaction, txHash []byte) {
	t.ChainID = tx.ChainId

	t.Gas = tx.Gas
	t.GasPrice = tx.GasPrice
	t.PublicKey = misc.BytesToHexStr(tx.Pk)
	t.Signature = misc.BytesToHexStr(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = misc.BytesToHexStr(txHash)
	t.TransactionType = "stake"

	t.Amount = tx.GetStake().Amount
}

func (t *PlainStakeTransaction) ToStakeTransactionObject() (*transactions.Stake, error) {
	pk, err := misc.HexStrToBytes(t.PublicKey)
	if err != nil {
		return nil, err
	}

	stakeTx := transactions.NewStake(
		t.ChainID,
		t.Amount,
		t.Gas,
		t.GasPrice,
		t.Nonce,
		pk)

	if stakeTx == nil {
		return nil, errors.New("error parsing stake transaction")
	}

	stakeTx.PBData().Signature, err = misc.HexStrToBytes(t.Signature)
	stakeTx.PBData().Hash, err = misc.HexStrToBytes(t.TransactionHash)

	return stakeTx, err
}
