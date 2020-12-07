package view

import (
	"encoding/hex"
	"errors"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainTransferTransaction struct {
	NetworkID		uint64 `json:"networkID"`
	MasterAddress   string `json:"masterAddress"`
	Fee             uint64 `json:"fee"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           uint64 `json:"nonce"`
	TransactionHash string `json:"transactionHash"`
	TransactionType string `json:"transactionType"`

	AddressesTo []string `json:"addressesTo"`
	Amounts     []uint64 `json:"amounts"`
	SlavePks    []string `json:"slavePks"`
	Message     string   `json:"message"`
}

func (t *PlainTransferTransaction) TransactionFromPBData(tx *protos.Transaction, txHash []byte) {
	t.NetworkID = tx.NetworkId
	if tx.MasterAddr != nil {
		t.MasterAddress = misc.Bin2Qaddress(tx.MasterAddr)
	}
	t.Fee = tx.Fee
	t.PublicKey = hex.EncodeToString(tx.Pk)
	t.Signature = hex.EncodeToString(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = hex.EncodeToString(txHash)
	t.TransactionType = "transfer"

	t.AddressesTo = misc.Bin2QAddresses(tx.GetTransfer().AddrsTo)
	t.Amounts = tx.GetTransfer().Amounts
	for _, slavePk := range tx.GetTransfer().SlavePks {
		t.SlavePks = append(t.SlavePks, hex.EncodeToString(slavePk))
	}
	t.Message = string(tx.GetTransfer().Message)
}

func (t *PlainTransferTransaction) ToTransferTransactionObject() (*transactions.Transfer, error) {
	addrsTo := misc.StringAddressToBytesArray(t.AddressesTo)
	xmssPK := misc.HStr2Bin(t.PublicKey)
	var masterAddr []byte
	var slavePks [][]byte
	var message []byte

	if len(t.MasterAddress) > 0 {
		masterAddr = misc.HStr2Bin(t.MasterAddress)
	}

	if len(t.SlavePks) > 0 {
		for _, slavePk := range t.SlavePks {
			slavePks = append(slavePks, misc.HStr2Bin(slavePk))
		}
	}

	if len(t.Message) > 0 {
		message = []byte(t.Message)
	}

	transferTx := transactions.NewTransfer(
		t.NetworkID,
		addrsTo,
		t.Amounts,
		t.Fee,
		slavePks,
		message,
		t.Nonce,
		xmssPK,
		masterAddr)

	if transferTx == nil {
		return nil, errors.New("error parsing transfer transaction")
	}

	transferTx.PBData().Signature = misc.HStr2Bin(t.Signature)

	return transferTx, nil
}