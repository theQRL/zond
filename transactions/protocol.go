package transactions

import (
	"bytes"
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"google.golang.org/protobuf/encoding/protojson"
)

type ProtocolTransactionInterface interface {
	Size() int

	PBData() *protos.ProtocolTransaction

	SetPBData(*protos.ProtocolTransaction)

	Type() TxType

	ChainID() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	Hash() common.Hash

	TxHash(signingHash common.Hash) common.Hash

	FromPBData(pbData *protos.ProtocolTransaction) //Set return type

	GetSigningHash(common.Hash) common.Hash

	GetUnsignedHash() common.Hash

	Sign(dilithium *dilithium.Dilithium, message []byte)

	ApplyStateChanges(stateContext *state.StateContext) error

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

func (tx *ProtocolTransaction) Type() TxType {
	return GetProtocolTransactionType(tx.pbData)
}

func (tx *ProtocolTransaction) ChainID() uint64 {
	return tx.pbData.ChainId
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

func (tx *ProtocolTransaction) Hash() common.Hash {
	return common.BytesToHash(tx.pbData.Hash)
}

func (tx *ProtocolTransaction) TxHash(signingHash common.Hash) common.Hash {
	return tx.GenerateTxHash(signingHash)
}

func (tx *ProtocolTransaction) SetNonce(n uint64) {
	tx.pbData.Nonce = n
}

func (tx *ProtocolTransaction) FromPBData(pbData *protos.ProtocolTransaction) {
	tx.pbData = pbData
}

func (tx *ProtocolTransaction) GetSigningHash() common.Hash {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) GetUnsignedHash() common.Hash {
	panic("Not Implemented")
}

func (tx *ProtocolTransaction) GenerateTxHash(hashableBytes common.Hash) common.Hash {
	tmp := new(bytes.Buffer)
	tmp.Write(hashableBytes[:])
	tmp.Write(tx.Signature())
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())
	outputHash := h.Sum(nil)

	var hash common.Hash
	copy(hash[:], outputHash)
	return hash
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
	pk := dilithium.GetPK()
	tx.pbData.Pk = pk[:]

	txHash := tx.TxHash(common.BytesToHash(message))
	tx.pbData.Hash = txHash[:]
}

func (tx *ProtocolTransaction) ApplyStateChanges(stateContext *state.StateContext) error {
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
