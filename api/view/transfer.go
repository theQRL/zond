package view

import (
	"encoding/hex"
	"errors"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/transactions"
)

type PlainTransferTransaction struct {
	NetworkID       uint64 `json:"networkID"`
	Gas             uint64 `json:"gas"`
	GasPrice        uint64 `json:"gasPrice"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           uint64 `json:"nonce"`
	TransactionHash string `json:"transactionHash"`
	TransactionType string `json:"transactionType"`

	To    string `json:"to"`
	Value uint64 `json:"value"`
	Data  string `json:"data"`
}

func (t *PlainTransferTransaction) TransactionFromPBData(tx *protos.Transaction, txHash []byte) {
	t.NetworkID = tx.NetworkId
	t.Gas = tx.Gas
	t.GasPrice = tx.GasPrice
	t.PublicKey = hex.EncodeToString(tx.Pk)
	t.Signature = hex.EncodeToString(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = hex.EncodeToString(txHash)
	t.TransactionType = "transfer"

	t.To = hex.EncodeToString(tx.GetTransfer().To)
	t.Value = tx.GetTransfer().Value
	t.Data = hex.EncodeToString(tx.GetTransfer().Data)
}

func (t *PlainTransferTransaction) ToTransferTransactionObject() (*transactions.Transfer, error) {
	to, err := hex.DecodeString(t.To)
	if err != nil {
		return nil, err
	}

	pk, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		return nil, err
	}

	data, err := hex.DecodeString(t.Data)
	if err != nil {
		return nil, err
	}

	transferTx := transactions.NewTransfer(
		t.NetworkID,
		to,
		t.Value,
		t.Gas,
		t.GasPrice,
		data,
		t.Nonce,
		pk)

	if transferTx == nil {
		return nil, errors.New("error parsing transfer transaction")
	}

	transferTx.PBData().Signature, err = hex.DecodeString(t.Signature)

	return transferTx, err
}
