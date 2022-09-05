package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
)

type Attest struct {
	ProtocolTransaction
}

func (tx *Attest) GetSigningHash(partialBlockSigningHash common.Hash) common.Hash {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, partialBlockSigningHash)
	binary.Write(tmp, binary.BigEndian, tx.ChainID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	h := sha256.New()
	h.Write(tmp.Bytes())

	outputHash := h.Sum(nil)

	var hash common.Hash
	copy(hash[:], outputHash)
	return hash
}

func (tx *Attest) validateData(stateContext *state.StateContext) bool {
	return true
}

func (tx *Attest) Validate(stateContext *state.StateContext) bool {

	return true
}

func (tx *Attest) ApplyStateChanges(stateContext *state.StateContext) error {
	/*
		CoinBase transaction already adding reward to all attestors,
		so nothing to do in this function
	*/
	return nil
}

func NewAttest(chainID uint64, nonce uint64) *Attest {
	tx := &Attest{}

	tx.pbData = &protos.ProtocolTransaction{}
	tx.pbData.Type = &protos.ProtocolTransaction_Attest{Attest: &protos.Attest{}}

	tx.pbData.ChainId = chainID

	tx.pbData.Nonce = nonce

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}

func AttestTransactionFromPBData(pbData *protos.ProtocolTransaction) *Attest {
	switch pbData.Type.(type) {
	case *protos.ProtocolTransaction_Attest:
		return &Attest{
			ProtocolTransaction{
				pbData: pbData,
			},
		}
	default:
		panic("pbData is not a attest transaction")
	}
}
