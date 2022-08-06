package view

import (
	"encoding/hex"
	"errors"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/transactions"
)

type PlainStakeTransaction struct {
	NetworkID       uint64 `json:"networkID"`
	Gas             uint64 `json:"gas"`
	GasPrice        uint64 `json:"gasPrice"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           uint64 `json:"nonce"`
	TransactionHash string `json:"transactionHash"`
	TransactionType string `json:"transactionType"`

	Amount uint64 `json:"amount"`
}

func (t *PlainStakeTransaction) TransactionFromPBData(tx *protos.Transaction, txHash []byte) {
	t.NetworkID = tx.NetworkId

	t.Gas = tx.Gas
	t.GasPrice = tx.GasPrice
	t.PublicKey = hex.EncodeToString(tx.Pk)
	t.Signature = hex.EncodeToString(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = hex.EncodeToString(txHash)
	t.TransactionType = "stake"

	t.Amount = tx.GetStake().Amount
}

func (t *PlainStakeTransaction) ToStakeTransactionObject() (*transactions.Stake, error) {
	pk, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		return nil, err
	}

	stakeTx := transactions.NewStake(
		t.NetworkID,
		t.Amount,
		t.Gas,
		t.GasPrice,
		t.Nonce,
		pk)

	if stakeTx == nil {
		return nil, errors.New("error parsing stake transaction")
	}

	stakeTx.PBData().Signature, err = hex.DecodeString(t.Signature)

	return stakeTx, err
}
