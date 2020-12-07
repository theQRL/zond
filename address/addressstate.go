package address

import (
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type AddressState struct {
	pbData *protos.AddressState
}

func (a *AddressState) PBData() *protos.AddressState {
	return a.pbData
}

func (a *AddressState) Address() []byte {
	return a.pbData.Address
}

func (a *AddressState) Height() uint64 {
	return uint64(a.pbData.Address[1] << 1)
}

func (a *AddressState) Nonce() uint64 {
	return a.pbData.Nonce
}

func (a *AddressState) Balance() uint64 {
	return a.pbData.Balance
}

func (a *AddressState) StakeBalance() uint64 {
	return a.pbData.StakeBalance
}

func (a *AddressState) SetBalance(balance uint64) {
	a.pbData.Balance = balance
}

func (a *AddressState) AddBalance(balance uint64) {
	a.pbData.Balance += balance
}

func (a *AddressState) SubtractBalance(balance uint64) {
	a.pbData.Balance -= balance
}

func (a *AddressState) LockStakeBalance(balance uint64) {
	a.pbData.Balance -= balance
	a.pbData.StakeBalance += balance
}

func (a *AddressState) ReleaseStakeBalance(balance uint64) {
	a.pbData.StakeBalance -= balance
	a.pbData.Balance += balance
}

func (a *AddressState) IncreaseNonce() {
	a.pbData.Nonce++
}

func (a *AddressState) DecreaseNonce() {
	a.pbData.Nonce--
}

func (a *AddressState) Serialize() ([]byte, error) {
	return proto.Marshal(a.pbData)
}

func (a *AddressState) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, a.pbData)
}

func (a *AddressState) Commit(b *bbolt.Bucket) error {
	data, err := a.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetAddressStateKey(a.Address()), data)
}

func NewAddressState(address []byte, nonce uint64, balance uint64) *AddressState {
	a := &AddressState{&protos.AddressState{}}
	a.pbData.Address = address
	a.pbData.Nonce = nonce
	a.pbData.Balance = balance

	return a
}

func GetAddressState(db *db.DB, address []byte, lastBlockHeaderHash []byte,
	finalizedHeaderHash []byte) (*AddressState, error) {
	key := GetAddressStateKey(address)

	data, err := metadata.GetDataByBucket(db, key, lastBlockHeaderHash, finalizedHeaderHash)
	if err != nil {
		return NewAddressState(address, 0, 0), nil
	}
	a := &AddressState{
		pbData: &protos.AddressState{},
	}
	return a, a.DeSerialize(data)
}

func GetAddressStateKey(address []byte) []byte {
	return []byte(fmt.Sprintf("ADDRESS-%s", hex.EncodeToString(address)))
}
