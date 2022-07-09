package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"google.golang.org/protobuf/encoding/protojson"
	"reflect"
)

type CoreTransaction interface {
	Size() int

	Type()

	NetworkID() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	TxHash(signingHash []byte) []byte

	ApplyStateChanges(stateContext *state.StateContext) error

	SetAffectedAddress(stateContext *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool
}

type TransactionInterface interface {
	Size() int

	PBData() *protos.Transaction

	SetPBData(*protos.Transaction)

	Type()

	NetworkID() uint64

	MasterAddr() []byte

	Fee() uint64

	Nonce() uint64

	PK() []byte

	Signature() []byte

	TxHash(signingHash []byte) []byte

	AddrFrom() []byte

	AddrFromPK() string

	OTSIndex() uint64

	FromPBData(pbData *protos.Transaction) //Set return type

	GetSlave() []byte

	GetSigningHash() []byte

	Sign(xmss *xmss.XMSS, message []byte)

	ApplyStateChanges(stateContext *state.StateContext) error

	applyStateChangesForPK(stateContext *state.StateContext) error

	SetAffectedAddress(stateContext *state.StateContext) error

	validateData(stateContext *state.StateContext) bool

	Validate(stateContext *state.StateContext) bool

	ValidateSlave(stateContext *state.StateContext) bool

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

func (tx *Transaction) Type() {
	//return tx.pbData.type.(type)
}

func (tx *Transaction) NetworkID() uint64 {
	return tx.pbData.NetworkId
}

func (tx *Transaction) MasterAddr() []byte {
	return tx.pbData.MasterAddr
}

func (tx *Transaction) Fee() uint64 {
	return tx.pbData.Fee
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

func (tx *Transaction) TxHash(signingHash []byte) []byte {
	return tx.GenerateTxHash(signingHash)
}

func (tx *Transaction) SetNonce(n uint64) {
	tx.pbData.Nonce = n
}

func (tx *Transaction) AddrFrom() []byte {
	if tx.MasterAddr() != nil {
		return tx.MasterAddr()
	}

	address := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(tx.PK()))
	return address[:]
}

func (tx *Transaction) AddrFromPK() string {
	address := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(tx.PK()))
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
	address := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(tx.PK()))

	if !reflect.DeepEqual(address, tx.AddrFrom()) {
		return address[:]
	}

	return nil
}

func (tx *Transaction) GetSigningHash() []byte {
	panic("Not Implemented")
}

func (tx *Transaction) GenerateTxHash(signingHash []byte) []byte {
	tmp := new(bytes.Buffer)
	tmp.Write(signingHash)
	tmp.Write(tx.Signature())
	tmp.Write(tx.PK())

	h := sha256.New()
	h.Write(tmp.Bytes())
	return h.Sum(nil)
}

func (tx *Transaction) Sign(x *xmss.XMSS, message []byte) {
	signature, err := x.Sign(message)
	if err != nil {
		panic("Failed To Sign")
	}
	tx.pbData.Signature = signature

}

func (tx *Transaction) applyStateChangesForPK(stateContext *state.StateContext) error {
	a, err := stateContext.GetAddressStateByPK(tx.PK())
	if err != nil {
		return err
	}
	a.IncreaseNonce()
	a.SubtractBalance(tx.Fee())
	stateContext.AddTransactionFee(tx.Fee())
	// TODO: Set Ots Key

	//if _, ok := addressesState[addrFromPK]; ok {
	//	//if hex.EncodeToString(tx.AddrFrom()) != addrFromPK {
	//	//	addressesState[addrFromPK].AppendTransactionHash(tx.Txhash())
	//	//}
	//	//if tx.OtsKey() >= tx.config.Dev.MaxOTSTracking {
	//	//	addressesState[addrFromPK].AppendTransactionHash(tx.Txhash())
	//	//}
	//	addressesState[addrFromPK].IncreaseNonce()
	//	addressesState[addrFromPK].SetOTSKey(tx.OtsIndex())
	//}
	return nil
}

func (tx *Transaction) ApplyStateChanges(stateContext *state.StateContext) error {
	panic("Not Implemented")
}

func (tx *Transaction) SetAffectedAddress(stateContext *state.StateContext) error {
	err := stateContext.PrepareAddressState(hex.EncodeToString(tx.AddrFrom()))
	if err != nil {
		return err
	}
	address := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(tx.PK()))
	err = stateContext.PrepareAddressState(hex.EncodeToString(address[:]))
	return err
}

func (tx *Transaction) validateData(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *Transaction) Validate(stateContext *state.StateContext) bool {
	panic("Not Implemented")
}

func (tx *Transaction) ValidateSlave(stateContext *state.StateContext) bool {
	masterAddr := tx.MasterAddr()
	slavePK := tx.PK()
	if len(masterAddr) == 0 {
		return true
	}
	addrFromPK := xmss.GetXMSSAddressFromPK(misc.UnSizedPKToSizedPK(tx.PK()))

	if reflect.DeepEqual(tx.MasterAddr(), addrFromPK) {
		log.Warn("Matching master_addr field and address from PK")
		return false
	}

	slaveMetaData := stateContext.GetSlaveState(hex.EncodeToString(masterAddr), hex.EncodeToString(slavePK))
	if slaveMetaData == nil {
		return false
	}
	if len(slaveMetaData.TxHash()) == 0 {
		return false
	}

	return true
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
