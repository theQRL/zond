package transactions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/misc"
)

type Transfer struct {
	Transaction
}

func (tx *Transfer) AddrsTo() [][]byte {
	return tx.pbData.GetTransfer().AddrsTo
}

func (tx *Transfer) Amounts() []uint64 {
	return tx.pbData.GetTransfer().Amounts
}

func (tx *Transfer) SlavePKs() [][]byte {
	return tx.pbData.GetTransfer().SlavePks
}

func (tx *Transfer) Message() []byte {
	return tx.pbData.GetTransfer().Message
}

func (tx *Transfer) TotalAmounts() uint64 {
	totalAmount := uint64(0)
	for _, amount := range tx.Amounts() {
		totalAmount += amount
	}
	return totalAmount
}

func (tx *Transfer) GetSigningHash() []byte {
	tmp := new(bytes.Buffer)
	binary.Write(tmp, binary.BigEndian, tx.NetworkID())
	tmp.Write(tx.MasterAddr())
	binary.Write(tmp, binary.BigEndian, tx.Fee())
	binary.Write(tmp, binary.BigEndian, tx.Nonce())

	for i := range tx.AddrsTo() {
		tmp.Write(tx.AddrsTo()[i])
		binary.Write(tmp, binary.BigEndian, tx.Amounts()[i])
	}

	for _, slavePK := range tx.SlavePKs() {
		tmp.Write(slavePK)
	}

	tmp.Write(tx.Message())

	tmpTXHash := goqrllib.Sha2_256(misc.BytesToUCharVector(tmp.Bytes()))

	return misc.UCharVectorToBytes(tmpTXHash)
}

func (tx *Transfer) validateData(stateContext *state.StateContext) bool {
	txHash := tx.TxHash(tx.GetSigningHash())

	// TODO: Common check that the length of all bytes field such as addressfrom, slavepks, tx hash are of even length
	for _, amount := range tx.Amounts() {
		if amount < 0 {
			log.Warn("Transfer [%s] Invalid Amount = %d", misc.Bin2HStr(txHash), amount)
			return false
		}
	}

	if tx.Fee() < 0 {
		log.Warn("Transfer [%s] Invalid Fee = %d", misc.Bin2HStr(txHash), tx.Fee)
		return false
	}

	addressState, err := stateContext.GetAddressState(misc.Bin2HStr(tx.AddrFrom()))
	if err != nil {
		log.Warn("Transfer [%s] Address %s missing into state context", tx.AddrFrom())
		return false
	}

	if tx.Nonce() != addressState.Nonce() + 1 {
		log.Warn(fmt.Sprintf("Transfer [%s] Invalid Nonce %d, Expected Nonce %d",
			misc.Bin2HStr(txHash), tx.Nonce(), addressState.Nonce() + 1))
		return false
	}

	balance := addressState.Balance()
	if balance < tx.TotalAmounts() + tx.Fee() {
		log.Warn("Insufficient balance",
			"txhash", misc.Bin2HStr(txHash),
			"balance", balance,
			"fee", tx.Fee())
		return false
	}

	// TODO: Max number of destination addresses allowed per transaction
	//if len(tx.AddrsTo()) > int(tx.config.Dev.Transaction.MultiOutputLimit) {
	//	log.Warn("[Transfer] Number of addresses exceeds max limit'")
	//	log.Warn(">> Length of addrsTo %s", len(tx.AddrsTo()))
	//	log.Warn(">> Length of amounts %s", len(tx.Amounts()))
	//	return false
	//}

	if len(tx.AddrsTo()) != len(tx.Amounts()) {
		log.Warn("[Transfer] Mismatch number of addresses to & amounts")
		log.Warn(">> Length of addrsTo %s", len(tx.AddrsTo()))
		log.Warn(">> Length of amounts %s", len(tx.Amounts()))
		return false
	}

	// TODO: Move to some common validation
	if !misc.IsValidAddress(tx.AddrFrom()) {
		log.Warn("[Transfer] Invalid address addr_from: %s", tx.AddrFrom())
		return false
	}

	for _, addrTo := range tx.AddrsTo() {
		if !misc.IsValidAddress(addrTo) {
			log.Warn("[Transfer] Invalid address addr_to: %s", tx.AddrsTo())
			return false
		}
	}

	slavePksLen := len(tx.SlavePKs())
	// TODO: Move 100 to the config
	if slavePksLen > 100 {
		log.Warn("Slave Public Keys length beyond length limit: %d", slavePksLen)
		return false
	}

	for _, slavePK := range tx.SlavePKs() {
		binAddress := misc.PK2BinAddress(slavePK)
		if !misc.IsValidAddress(binAddress) {
			log.Warn("Slave public key %s is invalid", misc.Bin2HStr(slavePK))
			return false
		}
		slaveState := stateContext.GetSlaveState(misc.Bin2HStr(tx.AddrFrom()), misc.Bin2HStr(slavePK))
		if slaveState != nil {
			log.Warn("Slave public key %s already exist for address %s", slavePK, tx.AddrFrom())
			return false
		}
	}

	msgLen := len(tx.Message())
	// TODO: Move 100 to the config
	if msgLen > 100 {
		log.Warn("Message length beyond message length limit: %d", msgLen)
		return false
	}

	return true
}

func (tx *Transfer) Validate(stateContext *state.StateContext) bool {
	/*
	TODO:
	1. Validate OTS Key Reuse
	2. Validate Slave [ done ]
	3. Validate Data in the fields [ done ]
	4. Validate Signature [done]
	 */
	txHash := tx.TxHash(tx.GetSigningHash())

	if !tx.ValidateSlave(stateContext) {
		return false
	}

	if !tx.validateData(stateContext) {
		log.Warn("Custom validation failed")
		return false
	}

	// TODO: Reject any transaction having destination to coinbase address
	// TODO: Reject any transaction having master address to coinbase address
	//if reflect.DeepEqual(tx.config.Dev.Genesis.CoinbaseAddress, tx.PK()) || reflect.DeepEqual(tx.config.Dev.Genesis.CoinbaseAddress, tx.MasterAddr()) {
	//	log.Warn("Coinbase Address only allowed to do Coinbase Transaction")
	//	return false
	//}

	expectedTransactionHash := tx.GenerateTxHash(tx.GetSigningHash())

	// TODO: Move to common function
	if !reflect.DeepEqual(expectedTransactionHash, txHash) {
		log.Warn("Invalid Transaction hash",
			"Expected Transaction hash", misc.Bin2HStr(expectedTransactionHash),
			"Found Transaction hash", misc.Bin2HStr(txHash))
		return false
	}

	// XMSS Signature Verification
	if !crypto.XMSSVerify(tx.GetSigningHash(), tx.Signature(), tx.PK()) {
		log.Warn("XMSS Verification Failed")
		return false
	}
	return true
}

func (tx *Transfer) ApplyStateChanges(stateContext *state.StateContext) error {
	if err := tx.applyStateChangesForPK(stateContext); err != nil {
		return err
	}

	addrState, err := stateContext.GetAddressState(misc.Bin2HStr(tx.AddrFrom()))
	if err != nil {
		return err
	}
	total := tx.TotalAmounts() + tx.Fee()
	addrState.SubtractBalance(total)

	addrsTo := tx.AddrsTo()
	amounts := tx.Amounts()
	for index := range addrsTo {
		addrTo := addrsTo[index]
		amount := amounts[index]

		addrState, err := stateContext.GetAddressState(misc.Bin2HStr(addrTo))
		if err != nil {
			return err
		}
		addrState.AddBalance(amount)
	}
	return nil
}

func (tx *Transfer) SetAffectedAddress(stateContext *state.StateContext) error {
	err := stateContext.PrepareAddressState(misc.Bin2HStr(tx.AddrFrom()))
	if err != nil {
		return err
	}
	addrFromPK := misc.PK2BinAddress(tx.PK())
	err = stateContext.PrepareAddressState(misc.Bin2HStr(addrFromPK))
	if err != nil {
		return err
	}

	for _, address := range tx.AddrsTo() {
		err = stateContext.PrepareAddressState(misc.Bin2HStr(address))
		if err != nil {
			return err
		}
	}

	for _, slavePKs := range tx.SlavePKs() {
		err = stateContext.PrepareSlaveMetaData(misc.Bin2HStr(tx.AddrFrom()), misc.Bin2HStr(slavePKs))
		if err != nil {
			return err
		}
	}

	if err := stateContext.PrepareOTSIndexMetaData(misc.Bin2HStr(addrFromPK), tx.OTSIndex()); err != nil {
		return err
	}

	return err
}

func NewTransfer(addrsTo [][]byte, amounts []uint64, fee uint64,
	slavesPK [][]byte, message []byte, xmssPK []byte, masterAddr []byte) *Transfer {
	tx := &Transfer{}

	tx.pbData = &protos.Transaction{}
	tx.pbData.Type = &protos.Transaction_Transfer{Transfer: &protos.Transfer{}}

	if masterAddr != nil {
		tx.pbData.MasterAddr = masterAddr
	}

	tx.pbData.Pk = xmssPK
	tx.pbData.Fee = fee
	transferPBData := tx.pbData.GetTransfer()
	transferPBData.AddrsTo = addrsTo
	transferPBData.Amounts = amounts
	transferPBData.Message = message

	for _, slavePK := range slavesPK {
		transferPBData.SlavePks = append(transferPBData.SlavePks, slavePK)
	}

	// TODO: Pass StateContext
	if !tx.Validate(nil) {
		return nil
	}

	return tx
}
