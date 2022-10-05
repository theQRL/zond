package view

import (
	"errors"

	"github.com/theQRL/zond/common/hexutil"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/transactions"
)

type PlainTransferTransaction struct {
	ChainID         uint64 `json:"chainID"`
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
	t.ChainID = tx.ChainId
	t.Gas = tx.Gas
	t.GasPrice = tx.GasPrice
	t.PublicKey = misc.BytesToHexStr(tx.Pk)
	t.Signature = misc.BytesToHexStr(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = misc.BytesToHexStr(txHash)
	t.TransactionType = "transfer"

	t.To = misc.BytesToHexStr(tx.GetTransfer().To)
	t.Value = tx.GetTransfer().Value
	t.Data = misc.BytesToHexStr(tx.GetTransfer().Data)
}

type PlainTransferTransactionRPC struct {
	ChainID   string `json:"chainId"`
	Gas       string `json:"gas"`
	GasPrice  string `json:"gasPrice"`
	PublicKey string `json:"pk"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
	To        string `json:"to"`
	Value     string `json:"value"`
}

func (t *PlainTransferTransactionRPC) TransactionFromPBData(tx *protos.Transaction) {
	t.ChainID = hexutil.EncodeUint64(tx.GetChainId()) //tx.ChainId
	t.Gas = hexutil.EncodeUint64(tx.GetGas())
	t.GasPrice = hexutil.EncodeUint64(tx.GetGasPrice())
	t.PublicKey = misc.BytesToHexStr(tx.GetPk())
	t.Signature = misc.BytesToHexStr(tx.GetSignature())
	t.Nonce = hexutil.EncodeUint64(tx.GetNonce())
	t.To = misc.BytesToHexStr(tx.GetTransfer().GetTo())
	t.Value = hexutil.EncodeUint64(tx.GetTransfer().GetValue())
}

func (t *PlainTransferTransaction) ToTransferTransactionObject() (*transactions.Transfer, error) {
	to, err := misc.HexStrToBytes(t.To)
	if err != nil {
		return nil, err
	}

	pk, err := misc.HexStrToBytes(t.PublicKey)
	if err != nil {
		return nil, err
	}

	data, err := misc.HexStrToBytes(t.Data)
	if err != nil {
		return nil, err
	}

	transferTx := transactions.NewTransfer(
		t.ChainID,
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

	transferTx.PBData().Signature, err = misc.HexStrToBytes(t.Signature)
	transferTx.PBData().Hash, err = misc.HexStrToBytes(t.TransactionHash)

	return transferTx, err
}
