package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"google.golang.org/protobuf/encoding/protojson"
	"reflect"
)

type CoreTransaction interface {
	Size() int

	Type() TxType

	ChainID() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	Hash() common.Hash

	ApplyStateChanges(stateContext *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool
}

type TransactionInterface interface {
	Size() int

	PBData() *protos.Transaction

	SetPBData(*protos.Transaction)

	Type() TxType

	ChainID() uint64

	Gas() uint64

	GasPrice() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	Hash() common.Hash

	AddrFrom() common.Address

	AddrFromPK() string

	OTSIndex() uint64

	FromPBData(pbData *protos.Transaction) //Set return type

	GetSlave() []byte

	GetSigningHash() common.Hash

	SignXMSS(xmss *xmss.XMSS, signingHash common.Hash)
	SignDilithium(d *dilithium.Dilithium, signingHash common.Hash)

	ApplyStateChanges(stateContext *state.StateContext) error

	applyStateChangesForPK(stateContext *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool

	ApplyEpochMetaData(epochMetaData *metadata.EpochMetaData) error
}

type Transaction struct {
	pbData *protos.Transaction
}

func (tx *Transaction) Size() int {
	return proto.Size(tx.pbData)
}

func (tx *Transaction) PBData() *protos.Transaction {
	return tx.pbData
}

func (tx *Transaction) SetPBData(pbData *protos.Transaction) {
	tx.pbData = pbData
}

func (tx *Transaction) Type() TxType {
	return GetTransactionType(tx.pbData)
}

func (tx *Transaction) ChainID() uint64 {
	return tx.pbData.ChainId
}

func (tx *Transaction) GasPrice() uint64 {
	return tx.pbData.GasPrice
}

func (tx *Transaction) GasTipCap() uint64 {
	return tx.pbData.GasPrice
}

func (tx *Transaction) GasFeeCap() uint64 {
	return tx.pbData.GasPrice
}

func (tx *Transaction) Gas() uint64 {
	return tx.pbData.Gas
}

func (tx *Transaction) Nonce() uint64 {
	return tx.pbData.Nonce
}

func (tx *Transaction) PK() []byte {
	return tx.pbData.Pk
}

func (tx *Transaction) Signature() []byte {
	return tx.pbData.Signature
}

func (tx *Transaction) Hash() common.Hash {
	return common.BytesToHash(tx.pbData.Hash)
}

func (tx *Transaction) SetNonce(n uint64) {
	tx.pbData.Nonce = n
}

func (tx *Transaction) AddrFrom() common.Address {
	addr := misc.GetAddressFromUnSizedPK(tx.PK())
	if addr == (common.Address{}) {
		panic(fmt.Sprintf("invalid PK size %d | PK: %v ", len(tx.PK()), tx.PK()))
	}
	return addr
}

func (tx *Transaction) AddrFromPK() string {
	var address common.Address
	if len(tx.PK()) == xmss.ExtendedPKSize {
		address = misc.GetXMSSAddressFromUnSizedPK(tx.PK())
	} else if len(tx.PK()) == dilithium.PKSizePacked {
		address = misc.GetDilithiumAddressFromUnSizedPK(tx.PK())
	} else {
		panic(fmt.Sprintf("invalid PK size %d | PK: %v ", len(tx.PK()), tx.PK()))
	}

	return misc.BytesToHexStr(address[:])
}

func (tx *Transaction) OTSIndex() uint64 {
	return uint64(binary.BigEndian.Uint32(tx.pbData.Signature[0:4]))
}

func (tx *Transaction) GetOtsFromSignature(signature []byte) uint64 {
	return binary.BigEndian.Uint64(signature[0:8])
}

func (tx *Transaction) FromPBData(pbData *protos.Transaction) {
	tx.pbData = pbData
}

func (tx *Transaction) GetSlave() []byte {
	address := misc.GetXMSSAddressFromUnSizedPK(tx.PK())

	if !reflect.DeepEqual(address, tx.AddrFrom()) {
		return address[:]
	}

	return nil
}

func (tx *Transaction) GetSigningHash() common.Hash {
	panic("Not Implemented")
}

func (tx *Transaction) generateTxHash(signingHash common.Hash) common.Hash {
	tmp := new(bytes.Buffer)
	tmp.Write(signingHash[:])
	tmp.Write(tx.Signature())
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())

	output := h.Sum(nil)
	var hash common.Hash
	copy(hash[:], output)

	return hash
}

func (tx *Transaction) SignXMSS(x *xmss.XMSS, signingHash common.Hash) {
	signature, err := x.Sign(signingHash[:])
	if err != nil {
		panic("Failed To Sign")
	}
	tx.pbData.Signature = signature

	txHash := tx.generateTxHash(signingHash)
	tx.pbData.Hash = txHash[:]
}

func (tx *Transaction) SignDilithium(d *dilithium.Dilithium, signingHash common.Hash) {
	signature := d.Sign(signingHash[:])
	tx.pbData.Signature = signature

	txHash := tx.generateTxHash(signingHash)
	tx.pbData.Hash = txHash[:]
}

func (tx *Transaction) applyStateChangesForPK(stateContext *state.StateContext) error {
	//address := xmss.GetXMSSAddressFromPK(misc.UnSizedXMSSPKToSizedPK(tx.PK()))
	//stateContext.AccountDB().SetNonce(address, tx.Nonce())
	//
	//fee := tx.Gas() * tx.GasPrice()
	//stateContext.AccountDB().SubBalance(address, big.NewInt(int64(fee)))
	//stateContext.AddTransactionFee(fee)

	return nil
}

func (tx *Transaction) ApplyStateChanges(stateContext *state.StateContext) error {
	panic("Not Implemented")
}

func (tx *Transaction) validateData(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *Transaction) Validate(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *Transaction) ValidateExtendedCoinbase(blockNumber uint64) bool {
	panic("Not Implemented")
}

func (tx *Transaction) ApplyEpochMetaData(epochMetaData *metadata.EpochMetaData) error {
	panic("Not Implemented")
}

func (tx *Transaction) ToJSON() ([]byte, error) {
	return protojson.Marshal(tx.pbData)
}

func ProtoToTransaction(protoTX *protos.Transaction) TransactionInterface {
	var tx TransactionInterface
	switch protoTX.Type.(type) {
	case *protos.Transaction_Stake:
		tx = &Stake{}
	case *protos.Transaction_Transfer:
		tx = &Transfer{}
	}

	if tx != nil {
		tx.SetPBData(protoTX)
	}

	return tx
}
