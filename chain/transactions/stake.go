package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/theQRL/go-qrllib-crypto/helper"
	"github.com/theQRL/go-qrllib-crypto/xmss"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type Stake struct {
	Transaction
}

func (tx *Stake) DilithiumPKs() [][]byte {
	return tx.pbData.GetStake().DilithiumPks
}

func (tx *Stake) Stake() bool {
	return tx.pbData.GetStake().Stake
}

func (tx *Stake) GetSigningHash() []byte {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.NetworkID())
	tmp.Write(tx.MasterAddr())
	binary.Write(tmp, binary.BigEndian, tx.Fee())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	for _, dilithiumPK := range tx.DilithiumPKs() {
		tmp.Write(dilithiumPK)
	}
	binary.Write(tmp, binary.BigEndian, tx.Stake())

	h := sha256.New()
	h.Write(tmp.Bytes())

	return h.Sum(nil)
}

func (tx *Stake) validateData(stateContext *state.StateContext) bool {
	txHash := tx.TxHash(tx.GetSigningHash())

	if tx.Fee() < 0 {
		log.Warn(fmt.Sprintf("Stake [%s] Invalid Fee = %d", hex.EncodeToString(txHash), tx.Fee()))
		return false
	}

	addressState, err := stateContext.GetAddressState(hex.EncodeToString(tx.AddrFrom()))
	if err != nil {
		log.Warn(fmt.Sprintf("[Stake] Address %s missing into state context", tx.AddrFrom()))
		return false
	}

	if tx.Nonce() != addressState.Nonce() {
		log.Warn(fmt.Sprintf("Stake [%s] Invalid Nonce %d, Expected Nonce %d",
			hex.EncodeToString(txHash), tx.Nonce(), addressState.Nonce()))
		return false
	}

	balance := addressState.Balance()
	if balance < tx.Fee() {
		log.Warn("Insufficient balance",
			"txhash", hex.EncodeToString(txHash),
			"balance", balance,
			"fee", tx.Fee())
		return false
	}

	if tx.Stake() {
		requiredBalance := tx.Fee() + uint64(len(tx.DilithiumPKs())) * config.GetDevConfig().MinStakeAmount
		if balance < requiredBalance {
			log.Warn("Insufficient balance ",
				"txhash ", hex.EncodeToString(txHash),
				"balance ", balance,
				"required balance ", requiredBalance)
			return false
		}
	} else {
		requiredStakeBalance := uint64(0)
		for _, dilithiumPK := range tx.DilithiumPKs() {
			strDilithiumPK := hex.EncodeToString(dilithiumPK)
			dilithiumMetadata := stateContext.GetDilithiumState(strDilithiumPK)
			if dilithiumMetadata == nil {
				log.Error("DilithiumMetaData not found to for de-staking ", strDilithiumPK)
				return false
			}
			requiredStakeBalance += dilithiumMetadata.Balance()
		}
		if addressState.StakeBalance() < requiredStakeBalance {
			log.Error("Stake Balance is lower than the required stake balance ",
				"Stake Balance ", addressState.StakeBalance(),
				"Required Stake Balance ", requiredStakeBalance)
			return false
		}
	}

	lenDilithiumPKs := len(tx.DilithiumPKs())
	// TODO: Move 100 into config
	if lenDilithiumPKs > 100 {
		log.Warn("Number of Dilithium Public Keys beyond limit [%s] Length = %d",
			hex.EncodeToString(txHash), lenDilithiumPKs)
		return false
	}

	for _, dilithiumPK := range tx.DilithiumPKs() {
		strDilithiumPK := hex.EncodeToString(dilithiumPK)
		dilithiumMetaData := stateContext.GetDilithiumState(strDilithiumPK)
		if dilithiumMetaData != nil {
			if !reflect.DeepEqual(dilithiumMetaData.Address(), tx.AddrFrom()) {
				log.Warn("Dilithium Public Key is already associated with another QRL address")
				return false
			}
			if dilithiumMetaData.Stake() == tx.Stake() {
				log.Warn("Dilithium Public Key %s has already stake status %s",
					strDilithiumPK, lenDilithiumPKs)
				return false
			}
		}
	}

	if !helper.IsValidAddress(tx.AddrFrom()) {
		log.Warn("[Stake] Invalid address addr_from: %s", tx.AddrFrom())
		return false
	}

	return true
}

func (tx *Stake) Validate(stateContext *state.StateContext) bool {
	txHash := tx.TxHash(tx.GetSigningHash())

	if !tx.ValidateSlave(stateContext) {
		return false
	}

	if !tx.validateData(stateContext) {
		log.Warn("Custom validation failed")
		return false
	}

	//if reflect.DeepEqual(tx.config.Dev.Genesis.CoinbaseAddress, tx.PK()) || reflect.DeepEqual(tx.config.Dev.Genesis.CoinbaseAddress, tx.MasterAddr()) {
	//	log.Warn("Coinbase Address only allowed to do Coinbase Transaction")
	//	return false
	//}

	expectedTransactionHash := tx.GenerateTxHash(tx.GetSigningHash())

	// TODO: Move to common function
	if !reflect.DeepEqual(expectedTransactionHash, txHash) {
		log.Warn("Invalid Transaction hash",
			"Expected Transaction hash", hex.EncodeToString(expectedTransactionHash),
			"Found Transaction hash", hex.EncodeToString(txHash))
		return false
	}

	// XMSS Signature Verification
	if !xmss.XMSSVerify(tx.GetSigningHash(), tx.Signature(), tx.PK()) {
		log.Warn("XMSS Verification Failed")
		return false
	}
	return true
}

func (tx *Stake) ApplyStateChanges(stateContext *state.StateContext) error {
	txHash := tx.TxHash(tx.GetSigningHash())

	if err := tx.applyStateChangesForPK(stateContext); err != nil {
		return err
	}

	addrState, err := stateContext.GetAddressState(hex.EncodeToString(tx.AddrFrom()))
	if err != nil {
		return err
	}

	for _, dilithiumPK := range tx.DilithiumPKs() {
		strDilithiumPK := hex.EncodeToString(dilithiumPK)

		dilithiumState := stateContext.GetDilithiumState(strDilithiumPK)

		if dilithiumState == nil {
			dilithiumState = metadata.NewDilithiumMetaData(txHash, dilithiumPK, tx.AddrFrom(), tx.Stake())

			if err := stateContext.AddDilithiumMetaData(strDilithiumPK, dilithiumState); err != nil {
				return err
			}
		}
		dilithiumState.SetStake(tx.Stake())
		if tx.Stake() {
			balance := config.GetDevConfig().MinStakeAmount
			addrState.LockStakeBalance(balance)
			dilithiumState.AddBalance(balance)
		} else {
			// TODO: Release of stake balance must happen after certain epoch
			balance := dilithiumState.Balance()
			dilithiumState.SubtractBalance(balance)
			addrState.ReleaseStakeBalance(balance)
		}
	}

	return nil
}

func (tx *Stake) SetAffectedAddress(stateContext *state.StateContext) error {
	// Pre-load dilithium metadata, so that the stake flag value can be validated
	for _, dilithiumPK := range tx.DilithiumPKs() {
		_ = stateContext.PrepareDilithiumMetaData(hex.EncodeToString(dilithiumPK))
	}

	err := stateContext.PrepareAddressState(hex.EncodeToString(tx.AddrFrom()))
	if err != nil {
		return err
	}

	addrFromPK := hex.EncodeToString(helper.PK2BinAddress(tx.PK()))

	err = stateContext.PrepareOTSIndexMetaData(addrFromPK, tx.OTSIndex())
	if err != nil {
		return err
	}

	err = stateContext.PrepareAddressState(addrFromPK)
	return err
}

func (tx *Stake) ApplyEpochMetaData(epochMetaData *metadata.EpochMetaData) error {
	if tx.Stake() {
		epochMetaData.AddValidators(tx.DilithiumPKs())
	} else {
		epochMetaData.RemoveValidators(tx.DilithiumPKs())
	}
	return nil
}

func NewStake(networkID uint64, dilithiumPKs [][]byte, stake bool,
	fee uint64, nonce uint64, xmssPK []byte, masterAddr []byte) *Stake {
	tx := &Stake{}

	tx.pbData = &protos.Transaction{}
	tx.pbData.NetworkId = networkID
	tx.pbData.Type = &protos.Transaction_Stake{Stake: &protos.Stake{}}

	if masterAddr != nil {
		tx.pbData.MasterAddr = masterAddr
	}

	tx.pbData.Pk = xmssPK
	tx.pbData.Fee = fee
	tx.pbData.Nonce = nonce
	tx.pbData.GetStake().DilithiumPks = dilithiumPKs
	tx.pbData.GetStake().Stake = stake

	// TODO: Pass StateContext
	//if !tx.Validate(nil) {
	//	return nil
	//}

	return tx
}
