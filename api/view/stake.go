package view

import (
	"errors"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainStakeTransaction struct {
	NetworkID		uint64 `json:"networkID"`
	MasterAddress   string `json:"masterAddress"`
	Fee             uint64 `json:"fee"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           uint64 `json:"nonce"`
	TransactionHash string `json:"transactionHash"`
	TransactionType string `json:"transactionType"`

	DilithiumPks []string `json:"dilithiumPks"`
	Stake        bool     `json:"stake"`
}

func (t *PlainStakeTransaction) TransactionFromPBData(tx *protos.Transaction, txHash []byte) {
	t.NetworkID = tx.NetworkId
	if tx.MasterAddr != nil {
		t.MasterAddress = misc.Bin2Qaddress(tx.MasterAddr)
	}
	t.Fee = tx.Fee
	t.PublicKey = misc.Bin2HStr(tx.Pk)
	t.Signature = misc.Bin2HStr(tx.Signature)
	t.Nonce = tx.Nonce
	t.TransactionHash = misc.Bin2HStr(txHash)
	t.TransactionType = "stake"

	for _, dilithiumPk := range tx.GetStake().DilithiumPks {
		t.DilithiumPks = append(t.DilithiumPks, misc.Bin2HStr(dilithiumPk))
	}
	t.Stake = tx.GetStake().Stake
}

func (t *PlainStakeTransaction) ToStakeTransactionObject() (*transactions.Stake, error) {
	xmssPK := misc.HStr2Bin(t.PublicKey)
	var masterAddr []byte
	var dilithiumPks [][]byte

	if len(t.MasterAddress) > 0 {
		masterAddr = misc.HStr2Bin(t.MasterAddress)
	}

	for _, dilithiumPk := range t.DilithiumPks {
		dilithiumPks = append(dilithiumPks, misc.HStr2Bin(dilithiumPk))
	}

	stakeTx := transactions.NewStake(
		t.NetworkID,
		dilithiumPks,
		t.Stake,
		t.Fee,
		t.Nonce,
		xmssPK,
		masterAddr)

	if stakeTx == nil {
		return nil, errors.New("error parsing stake transaction")
	}

	stakeTx.PBData().Signature = misc.HStr2Bin(t.Signature)

	return stakeTx, nil
}