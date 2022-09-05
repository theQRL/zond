package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
)

type CoinBase struct {
	ProtocolTransaction
}

func (tx *CoinBase) BlockProposerReward() uint64 {
	return tx.pbData.GetCoinBase().GetBlockProposerReward()
}

func (tx *CoinBase) AttestorReward() uint64 {
	return tx.pbData.GetCoinBase().GetAttestorReward()
}

func (tx *CoinBase) FeeReward() uint64 {
	return tx.pbData.GetCoinBase().FeeReward
}

func (tx *CoinBase) TotalRewardExceptFeeReward(numberOfAttestors uint64) uint64 {
	totalAmount := tx.BlockProposerReward() + tx.AttestorReward()*numberOfAttestors
	return totalAmount
}

func (tx *CoinBase) GetSigningHash(blockSigningHash common.Hash) common.Hash {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, blockSigningHash)
	binary.Write(tmp, binary.BigEndian, tx.ChainID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())
	// PK is considered as Block proposer
	// and attestor need to sign the unsigned coinbase transaction with
	binary.Write(tmp, binary.BigEndian, tx.PK())

	binary.Write(tmp, binary.BigEndian, tx.BlockProposerReward())
	binary.Write(tmp, binary.BigEndian, tx.AttestorReward())

	h := sha256.New()
	h.Write(tmp.Bytes())

	outputHash := h.Sum(nil)

	var hash common.Hash
	copy(hash[:], outputHash)
	return hash
}

func (tx *CoinBase) GetUnsignedHash() common.Hash {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.ChainID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())
	// PK is considered as Block proposer
	// and attestor need to sign the unsigned coinbase transaction with
	binary.Write(tmp, binary.BigEndian, tx.PK())

	binary.Write(tmp, binary.BigEndian, tx.BlockProposerReward())
	binary.Write(tmp, binary.BigEndian, tx.AttestorReward())

	h := sha256.New()
	h.Write(tmp.Bytes())

	outputHash := h.Sum(nil)
	var hash common.Hash
	copy(hash[:], outputHash)

	return hash
}

func (tx *CoinBase) validateData(stateContext *state.StateContext) bool {
	return true
}

func (tx *CoinBase) Validate(stateContext *state.StateContext) bool {

	if !tx.validateData(stateContext) {
		log.Warn("Data validation failed")
		return false
	}

	return true
}

func (tx *CoinBase) ApplyStateChanges(stateContext *state.StateContext) error {
	return nil
}

func NewCoinBase(chainId uint64, blockProposerDilithiumPK []byte, blockProposerReward uint64,
	attestorReward uint64, feeReward uint64, nonce uint64) *CoinBase {
	tx := &CoinBase{}

	tx.pbData = &protos.ProtocolTransaction{}
	tx.pbData.Type = &protos.ProtocolTransaction_CoinBase{CoinBase: &protos.CoinBase{}}

	// TODO: Derive Network ID based on the current connected network
	tx.pbData.ChainId = chainId
	tx.pbData.Pk = blockProposerDilithiumPK
	//tx.pbData.MasterAddr = tx.config.Dev.Genesis.CoinBaseAddress

	cb := tx.pbData.GetCoinBase()
	cb.BlockProposerReward = blockProposerReward
	cb.AttestorReward = attestorReward
	cb.FeeReward = feeReward

	tx.pbData.Nonce = nonce

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}

func CoinBaseTransactionFromPBData(pbData *protos.ProtocolTransaction) *CoinBase {
	switch pbData.Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
		return &CoinBase{
			ProtocolTransaction{
				pbData: pbData,
			},
		}
	default:
		panic("pbData is not a coinbase transaction")
	}
}
