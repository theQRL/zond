package metadata

import (
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/db"
	"reflect"
)

func GetDataByBucket(db *db.DB, key []byte, headerHash common.Hash, finalizedHeaderHash common.Hash) ([]byte, error) {
	var data []byte
	var err error

	for !reflect.DeepEqual(headerHash, finalizedHeaderHash) {
		bucketName := GetBlockBucketName(headerHash)
		data, err = db.GetFromBucket(key, bucketName)
		if err == nil {
			break
		}
		blockMetaData, err := GetBlockMetaData(db, headerHash)
		if err != nil {
			return nil, err
		}
		headerHash = blockMetaData.ParentHeaderHash()
	}
	if data == nil {
		data, err = db.Get(key)
		if err != nil {
			return nil, err
		}
	}
	return data, err
}
