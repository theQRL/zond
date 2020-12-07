package transactions

import (
	"bytes"
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"google.golang.org/protobuf/encoding/protojson"
)

type ProtocolTransactionInterface interface {
	Size() int

	PBData() *protos.ProtocolTransaction

	SetPBData(*protos.ProtocolTransaction)

	Type()

	NetworkID() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	TxHash(signingHash []byte) []byte

	FromPBData(pbdata protos.ProtocolTransaction) //Set return type

	GetSigningHash([]byte) []byte

	GetUnsignedHash() []byte

	Sign(dilithium *dilithium.Dilithium, message []byte)

	ApplyStateChanges(stateContext *state.StateContext) error

	SetAffectedAddress(addressesState *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool
}

type ProtocolTransaction struct {
	pbData *protos.ProtocolTransaction
}

func (tx *ProtocolTransaction) Size() int {
	return proto.Size(tx.pbData)
}

func (tx *ProtocolTransaction) PBData() *protos.ProtocolTransaction {
	return tx.pbData
}

func (tx *ProtocolTransaction) SetPBData(pbData *protos.ProtocolTransaction) {
	tx.pbData = pbData
}

func (tx *ProtocolTransaction) Type() {
	//return tx.pbData.type.(type)
}

func (tx *ProtocolTransaction) NetworkID() uint64 {
	return tx.pbData.NetworkId
}

func (tx *ProtocolTransaction) Nonce() uint64 {
	return tx.pbData.Nonce
}

func (tx *ProtocolTransaction) PK() []byte {
	return tx.pbData.Pk
}

func (tx *ProtocolTransaction) Signature() []byte {
	return tx.pbData.Signature
}

func (tx *ProtocolTransaction) TxHash(signingHash []byte) []byte {
	return tx.GenerateTxHash(signingHash)
}


func (tx *ProtocolTransaction) SetNonce(n uint64) {
	tx.pbData.Nonce = n
}

func (tx *ProtocolTransaction) FromPBData(pbData protos.ProtocolTransaction) {
	tx.pbData = &pbData
}

func (tx *ProtocolTransaction) GetSigningHash() []byte {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) GetUnsignedHash() []byte {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) GenerateTxHash(hashableBytes []byte) []byte {
	tmp := new(bytes.Buffer)
	tmp.Write(hashableBytes)
	tmp.Write(tx.Signature())
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())
	return h.Sum(nil)
}

func (tx *ProtocolTransaction) GenerateUnSignedTxHash(hashableBytes []byte) []byte {
	tmp := new(bytes.Buffer)
	tmp.Write(hashableBytes)
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())

	return h.Sum(nil)
}

func (tx *ProtocolTransaction) Sign(dilithium *dilithium.Dilithium, message []byte) {
	tx.pbData.Signature = dilithium.Sign(message)
	tx.pbData.Pk = dilithium.PK()
}

func (tx *ProtocolTransaction) ApplyStateChanges(stateContext *state.StateContext) error {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) SetAffectedAddress(stateContext *state.StateContext) error {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) validateData(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) Validate(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) ValidateExtendedCoinbase(blockNumber uint64) bool {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) ToJSON() ([]byte, error) {
	return protojson.Marshal(tx.pbData)
}

func ProtoToProtocolTransaction(protoTX *protos.ProtocolTransaction) ProtocolTransactionInterface {
	var tx ProtocolTransactionInterface
	switch protoTX.Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
		tx = &CoinBase{}
	case *protos.ProtocolTransaction_Attest:
		tx = &Attest{}
	}

	if tx != nil {
		tx.SetPBData(protoTX)
	}

	return tx
}
