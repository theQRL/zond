package metadata

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type DilithiumMetaData struct {
	pbData *protos.DilithiumMetaData
}

func (d *DilithiumMetaData) TxHash() []byte {
	return d.pbData.TxHash
}

func (d *DilithiumMetaData) DilithiumPK() []byte {
	return d.pbData.DilithiumPk
}

func (d *DilithiumMetaData) Address() []byte {
	return d.pbData.Address
}

func (d *DilithiumMetaData) Stake() bool {
	return d.pbData.Stake
}

func (d *DilithiumMetaData) Balance() uint64 {
	return d.pbData.Balance
}

func (d *DilithiumMetaData) SetTxHash(txHash []byte) {
	d.pbData.TxHash = txHash
}

func (d *DilithiumMetaData) SetAddress(address []byte) {
	d.pbData.Address = address
}

func (d *DilithiumMetaData) SetStake(stake bool) {
	d.pbData.Stake = stake
}

func (d *DilithiumMetaData) AddBalance(balance uint64) {
	d.pbData.Balance += balance
}

func (d *DilithiumMetaData) SubtractBalance(balance uint64) {
	d.pbData.Balance -= balance
}

func (d *DilithiumMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(d.pbData)
}

func (d *DilithiumMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, d.pbData)
}

func (d *DilithiumMetaData) Commit(b *bbolt.Bucket) error {
	data, err := d.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetDilithiumMetaDataKey(d.DilithiumPK()), data)
}

func NewDilithiumMetaData(txHash []byte, dilithiumPK []byte, address []byte, stake bool) *DilithiumMetaData {
	pbData := &protos.DilithiumMetaData {
		TxHash: txHash,
		DilithiumPk: dilithiumPK,
		Address: address,
		Stake: stake,
	}
	return &DilithiumMetaData {
		pbData: pbData,
	}
}

func GetDilithiumMetaData(db *db.DB, dilithiumPK []byte,
	headerHash []byte, finalizedHeaderHash []byte) (*DilithiumMetaData, error) {
	key := GetDilithiumMetaDataKey(dilithiumPK)

	if len(dilithiumPK) == 0 {
		log.Error("DilithiumPK length cannot be 0")
		return nil, errors.New("DilithiumPK length cannot be 0")
	}

	data, err := GetDataByBucket(db, key, headerHash, finalizedHeaderHash)
	if err != nil {
		log.Error("Error loading DilithiumMetaData for key ", string(key), err)
		return nil, err
	}
	dm := &DilithiumMetaData{
		pbData: &protos.DilithiumMetaData{},
	}
	return dm, dm.DeSerialize(data)
}

func GetDilithiumMetaDataKey(dilithiumPK []byte) []byte {
	return []byte(fmt.Sprintf("DILITHIUM-META-DATA-%s", misc.Bin2HStr(dilithiumPK)))
}

func GetXMSSAddressFromDilithiumPK(db *db.DB, dilithiumPK []byte,
	headerHash []byte, finalizedHeaderHash []byte) ([]byte, error) {

	dm, err := GetDilithiumMetaData(db, dilithiumPK,
		headerHash, finalizedHeaderHash)
	if err != nil {
		log.Error("Failed to Get XMSSAdress from Dilithium PK")
		return nil, err
	}
	return dm.Address(), nil
}
