package transactions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"

	log "github.com/sirupsen/logrus"
)

type Attest struct {
	ProtocolTransaction
}

func (tx *Attest) GetSigningHash(partialBlockSigningHash []byte) []byte {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, partialBlockSigningHash)
	binary.Write(tmp, binary.BigEndian, tx.NetworkID())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	tmpTXHash := misc.NewUCharVector()
	tmpTXHash.AddBytes(tmp.Bytes())
	tmpTXHash.New(goqrllib.Sha2_256(tmpTXHash.GetData()))

	return tmpTXHash.GetBytes()
}

func (tx *Attest) validateData(stateContext *state.StateContext) bool {
	dilithiumMetaData := stateContext.GetDilithiumState(misc.Bin2HStr(tx.PK()))
	if dilithiumMetaData == nil {
		return false
	}
	if !dilithiumMetaData.Stake() {
		return false
	}

	if err := stateContext.ProcessAttestorsFlag(tx.PK()); err != nil {
		log.Error("Failed to Process Attest transaction for attestor ", misc.Bin2HStr(tx.PK()))
		log.Error("Reason: ", err.Error())
		return false
	}
	return true
}

func (tx *Attest) Validate(stateContext *state.StateContext) bool {
	messageSigned := tx.GetSigningHash(stateContext.PartialBlockSigningHash())
	txHash := tx.TxHash(messageSigned)

	if !dilithium.DilithiumVerify(tx.Signature(), tx.PK(), messageSigned) {
		log.Warn(fmt.Sprintf("Dilithium Signature Verification failed for Attest Txn %s",
			misc.Bin2HStr(txHash)))
		return false
	}

	if !tx.validateData(stateContext) {
		log.Warn("Data validation failed")
		return false
	}

	return true
}

func (tx *Attest) ApplyStateChanges(stateContext *state.StateContext) error {
	/*
	For each attest add Points to stateContext to validate the
	justification &finality
	 */
	return nil
}

func (tx *Attest) SetAffectedAddress(stateContext *state.StateContext) error {
	err := stateContext.PrepareDilithiumMetaData(misc.Bin2HStr(tx.PK()))
	if err != nil {
		return err
	}

	return err
}

func NewAttest(networkID uint64, coinBaseNonce uint64) *Attest {
	tx := &Attest{}

	tx.pbData = &protos.ProtocolTransaction{}
	tx.pbData.Type = &protos.ProtocolTransaction_Attest{Attest: &protos.Attest{}}

	// TODO: Derive Network ID based on the current connected network
	tx.pbData.NetworkId = networkID

	// TODO: Make nonce for CoinBase sequential, as this will not be sequential due to slotNumber
	tx.pbData.Nonce = coinBaseNonce

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}
