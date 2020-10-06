package metadata

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

type OTSIndexMetaData struct {
	pbData *protos.OTSIndexMetaData
}

func (o *OTSIndexMetaData) Address() []byte {
	return o.pbData.Address
}

func (o *OTSIndexMetaData) PageNumber() uint64 {
	return o.pbData.PageNumber
}

func (o *OTSIndexMetaData) Bitfield() [][]byte {
	return o.pbData.Bitfield
}

func (o *OTSIndexMetaData) Serialize() ([]byte, error) {
	return proto.Marshal(o.pbData)
}

func (o *OTSIndexMetaData) DeSerialize(data []byte) error {
	return proto.Unmarshal(data, o.pbData)
}

func (o *OTSIndexMetaData) IsOTSIndexUsed(otsIndex uint64) bool {
	// TODO: Move 8 to dev config + Add check if otsIndex beyond expected range (as max possible is 8192)
	otsIndex = otsIndex - o.PageNumber() * config.GetDevConfig().OTSBitFieldPerPage * 8
	offset := otsIndex >> 3
	relative := otsIndex % 8

	return ((o.pbData.Bitfield[offset][0] >> relative) & 1) == 1
}

func (o *OTSIndexMetaData) SetOTSIndex(otsIndex uint64) bool {
	if o.IsOTSIndexUsed(otsIndex) {
		return false
	}

	// TODO: Move 8 to dev config + Add check if otsIndex beyond expected range (as max possible is 8192)
	otsIndex = otsIndex - o.PageNumber() * config.GetDevConfig().OTSBitFieldPerPage * 8
	offset := otsIndex >> 3
	relative := otsIndex % 8
	o.pbData.Bitfield[offset][0] = o.pbData.Bitfield[offset][0] | (1 << relative)

	return true
}

func (o *OTSIndexMetaData) Commit(b *bbolt.Bucket) error {
	data, err := o.Serialize()
	if err != nil {
		return err
	}
	return b.Put(GetOTSIndexMetaDataKeyByPageNumber(o.Address(), o.PageNumber()), data)
}

func NewOTSIndexMetaData(address []byte, pageNumber uint64) *OTSIndexMetaData {
	pbData := &protos.OTSIndexMetaData {
		Address: address,
		PageNumber: pageNumber,
	}

	for i := 0; i < int(config.GetDevConfig().OTSBitFieldPerPage); i++ {
		pbData.Bitfield[i] = make([]byte, 8)
	}

	return &OTSIndexMetaData {
		pbData: pbData,
	}
}

func GetOTSIndexMetaData(db *db.DB, address []byte, otsIndex uint64,
	headerHash []byte, finalizedHeaderHash []byte) (*OTSIndexMetaData, error) {
	key := GetOTSIndexMetaDataKeyByOTSIndex(address, otsIndex)
	data, err := GetDataByBucket(db, key, headerHash, finalizedHeaderHash)

	// TODO: Check for error KeyNotFound
	// If error key not found then load with default data
	if err != nil {
		return NewOTSIndexMetaData(address,
			otsIndex / config.GetDevConfig().OTSBitFieldPerPage), nil
	}
	o := &OTSIndexMetaData{
		pbData: &protos.OTSIndexMetaData{},
	}
	return o, o.DeSerialize(data)
}

func GetOTSIndexMetaDataKeyByOTSIndex(address []byte, otsIndex uint64) []byte {
	pageNumber := otsIndex / config.GetDevConfig().OTSBitFieldPerPage

	return GetOTSIndexMetaDataKeyByPageNumber(address, pageNumber)
}

func GetOTSIndexMetaDataKeyByPageNumber(address []byte, pageNumber uint64) []byte {
	return []byte(fmt.Sprintf("OTS-INDEX-METADATA-%s-%d", address, pageNumber))
}
