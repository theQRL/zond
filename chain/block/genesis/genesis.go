package genesis

import (
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/address"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/block/genesis/devnet"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/protos"
	"go.etcd.io/bbolt"
)

func LoadPreState() (*protos.PreState, error) {
	jsonData, err := yaml.YAMLToJSON(devnet.DevnetGenesisData["prestate.yml"])
	if err != nil {
		return nil, err
	}

	preState := &protos.PreState{}
	err = jsonpb.UnmarshalString(string(jsonData), preState)
	if err != nil {
		return nil, err
	}

	return preState, nil
}

func ProcessPreState(preState *protos.PreState, b *block.Block, db *db.DB) error {
	m := metadata.NewMainChainMetaData(nil, 0,
		nil, 0)
	bm := metadata.NewBlockMetaData(nil, b.ParentHeaderHash(), 0)
	err := db.DB().Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		if err := m.Commit(b); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metadata.GetBlockBucketName(bm.HeaderHash())); err != nil {
			return err
		}
		return bm.Commit(b)
	})
	if err != nil {
		return err
	}

	return db.DB().Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		for _, addressBalance := range preState.AddressBalance {
			addressState := address.NewAddressState(addressBalance.Address,
				0, addressBalance.Balance)
			if err := addressState.Commit(b); err != nil {
				return err
			}
		}
		return nil
	})
}

func ProcessGenesisBlock(db *db.DB) error {
	jsonData, err := yaml.YAMLToJSON(devnet.DevnetGenesisData["genesis.yml"])
	if err != nil {
		return err
	}

	pbData := &protos.Block{}
	err = jsonpb.UnmarshalString(string(jsonData), pbData)
	if err != nil {
		return err
	}

	b := block.BlockFromPBData(pbData)

	preState, err := LoadPreState()
	if err != nil {
		log.Error("Failed to load PreState file")
		return nil
	}
	if err := ProcessPreState(preState, b, db); err != nil {
		log.Error("Failed to process pre-state")
		return err
	}

	return b.CommitGenesis(db, preState.AddressBalance[1].Address)
}
