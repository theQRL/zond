package view

import (
	"encoding/hex"
	"errors"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"strconv"
)

type PlainStakeTransaction struct {
	NetworkID       uint64 `json:"networkID"`
	MasterAddress   string `json:"masterAddress"`
	Fee             string `json:"fee"`
	PublicKey       string `json:"publicKey"`
	Signature       string `json:"signature"`
	Nonce           string `json:"nonce"`
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
	t.Fee = strconv.FormatUint(tx.Fee, 10)
	t.PublicKey = hex.EncodeToString(tx.Pk)
	t.Signature = hex.EncodeToString(tx.Signature)
	t.Nonce = strconv.FormatUint(tx.Nonce, 10)
	t.TransactionHash = hex.EncodeToString(txHash)
	t.TransactionType = "stake"

	for _, dilithiumPk := range tx.GetStake().DilithiumPks {
		t.DilithiumPks = append(t.DilithiumPks, hex.EncodeToString(dilithiumPk))
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

	fee, err := strconv.ParseUint(t.Fee, 10, 64)
	if err != nil {
		return nil, err
	}

	nonce, err := strconv.ParseUint(t.Nonce, 10, 64)
	if err != nil {
		return nil, err
	}

	stakeTx := transactions.NewStake(
		t.NetworkID,
		dilithiumPks,
		t.Stake,
		fee,
		nonce,
		xmssPK,
		masterAddr)

	if stakeTx == nil {
		return nil, errors.New("error parsing stake transaction")
	}

	stakeTx.PBData().Signature = misc.HStr2Bin(t.Signature)

	return stakeTx, nil
}
