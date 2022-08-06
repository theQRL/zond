package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"google.golang.org/protobuf/encoding/protojson"
	"math/big"
	"reflect"
)

type CoreTransaction interface {
	Size() int

	Type() uint8

	NetworkID() uint64

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

	Type() uint8

	NetworkID() uint64

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

	GetSigningHash() []byte

	SignXMSS(xmss *xmss.XMSS, message []byte)
	SignDilithium(d *dilithium.Dilithium, message []byte)

	ApplyStateChanges(stateContext *state.StateContext) error

	applyStateChangesForPK(stateContext *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool

	ApplyEpochMetaData(epochMetaData *metadata.EpochMetaData) error

	AsMessage(baseFee *big.Int) (types.Message, error)
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

func (tx *Transaction) Type() uint8 {
	switch tx.pbData.Type.(type) {
	case *protos.Transaction_Stake:
		return 2
	case *protos.Transaction_Transfer:
		return 3
	}
	panic("invalid transaction")
}

func (tx *Transaction) NetworkID() uint64 {
	return tx.pbData.NetworkId
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
	return tx.GenerateTxHash()
}

func (tx *Transaction) SetNonce(n uint64) {
	tx.pbData.Nonce = n
}

func (tx *Transaction) AddrFrom() common.Address {
	if len(tx.PK()) == xmss.ExtendedPKSize {
		return xmss.GetXMSSAddressFromPK(misc.UnSizedXMSSPKToSizedPK(tx.PK()))
	} else if len(tx.PK()) == dilithium.PKSizePacked {
		return dilithium.GetDilithiumAddressFromPK(misc.UnSizedDilithiumPKToSizedPK(tx.PK()))
	}
	panic(fmt.Sprintf("invalid PK size %d | PK: %v ", len(tx.PK()), tx.PK()))
}

func (tx *Transaction) AddrFromPK() string {
	var address common.Address
	if len(tx.PK()) == xmss.ExtendedPKSize {
		address = xmss.GetXMSSAddressFromPK(misc.UnSizedXMSSPKToSizedPK(tx.PK()))
	} else if len(tx.PK()) == dilithium.PKSizePacked {
		address = dilithium.GetDilithiumAddressFromPK(misc.UnSizedDilithiumPKToSizedPK(tx.PK()))
	} else {
		panic(fmt.Sprintf("invalid PK size %d | PK: %v ", len(tx.PK()), tx.PK()))
	}

	return hex.EncodeToString(address[:])
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
	address := xmss.GetXMSSAddressFromPK(misc.UnSizedXMSSPKToSizedPK(tx.PK()))

	if !reflect.DeepEqual(address, tx.AddrFrom()) {
		return address[:]
	}

	return nil
}

func (tx *Transaction) GetSigningHash() []byte {
	panic("Not Implemented")
}

func (tx *Transaction) GenerateTxHash() common.Hash {
	signingHash := tx.GetSigningHash()
	tmp := new(bytes.Buffer)
	tmp.Write(signingHash)
	tmp.Write(tx.Signature())
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())

	output := h.Sum(nil)
	var hash common.Hash
	copy(hash[:], output)

	return hash
}

func (tx *Transaction) SignXMSS(x *xmss.XMSS, message []byte) {
	signature, err := x.Sign(message)
	if err != nil {
		panic("Failed To Sign")
	}
	tx.pbData.Signature = signature
}

func (tx *Transaction) SignDilithium(d *dilithium.Dilithium, message []byte) {
	signature := d.Sign(message)
	tx.pbData.Signature = signature
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

func (tx *Transaction) AsMessage(baseFee *big.Int) (types.Message, error) {
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
