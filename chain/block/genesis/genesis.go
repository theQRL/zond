package genesis

import (
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/state"
	"go.etcd.io/bbolt"
	"io/ioutil"
	"path"
	"runtime"
)

func ProcessPreState(b *block.Block, db *db.DB) error {
	_, filename, _, _ := runtime.Caller(0)
	directory := path.Dir(filename) // Current Package path
	yamlData, err := ioutil.ReadFile(path.Join(directory, "prestate.yml"))

	if err != nil {
		return err
	}

	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return err
	}

	preState := &protos.PreState{}
	err = jsonpb.UnmarshalString(string(jsonData), preState)
	if err != nil {
		return err
	}

	m := metadata.NewMainChainMetaData(nil, 0,
		nil, 0)
	bm := metadata.NewBlockMetaData(nil, b.ParentHeaderHash(), 0)
	err = db.DB().Update(func(tx *bbolt.Tx) error {
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

	s, err := state.NewStateContext(db, 0, nil, nil,
		nil, nil, nil, nil)
	if err != nil {
		log.Error("Failed to create StateContext")
		return err
	}

	epochMetaData := metadata.NewEpochMetaData(0, nil, nil)

	for _, pbData := range preState.Transactions {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.ApplyEpochMetaData(epochMetaData); err != nil {
			log.Error("Failed to Apply EpochMetaData Changes")
			return err
		}
	}

	// TODO: Replace RandomSeed based on genesis config
	epochMetaData.AllotSlots(100, 0, b.ParentHeaderHash())

	data, err := epochMetaData.Serialize()
	if err != nil {
		log.Error("Failed to Serialize Epoch MetaData")
		return err
	}
	err = db.DB().Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		key := metadata.GetEpochMetaDataKey(0, []byte(""))
		return b.Put(key, data)
	})
	if err != nil {
		log.Error("Failed to commit EpochMetaData for Genesis Block")
		return err
	}

	for _, pbData := range preState.Transactions {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.SetAffectedAddress(s); err != nil {
			log.Error("Failed to SetAffectedAddress")
			return err
		}
	}

	for _, pbData := range preState.Transactions {
		tx := transactions.ProtoToTransaction(pbData)
		if err := tx.ApplyStateChanges(s); err != nil {
			log.Error("Failed to Apply State Changes")
			return err
		}
		addr, err := s.GetAddressState(tx.AddrFromPK())
		if err != nil {
			log.Error("[ProcessPreState] Error while loading address state")
			return err
		}
		addr.DecreaseNonce()
	}

	for _, addressBalance := range preState.AddressBalance {
		strAddress := misc.Bin2HStr(addressBalance.Address)
		if err := s.PrepareAddressState(strAddress); err != nil {
			return err
		}
		addressState, err := s.GetAddressState(strAddress)
		if err != nil {
			return err
		}
		addressState.AddBalance(addressBalance.Balance)
	}

	bytesBlock, err := b.Serialize()
	if err != nil {
		log.Error("Error while serializing Genesis Block")
		return err
	}

	return s.Commit(bytesBlock, true)
}

func ProcessGenesisBlock(db *db.DB) error {
	_, filename, _, _ := runtime.Caller(0)
	directory := path.Dir(filename) // Current Package path
	yamlData, err := ioutil.ReadFile(path.Join(directory, "genesis.yml"))

	if err != nil {
		return err
	}

	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return err
	}

	pbData := &protos.Block{}
	err = jsonpb.UnmarshalString(string(jsonData), pbData)
	if err != nil {
		return err
	}

	b := block.BlockFromPBData(pbData)

	if err := ProcessPreState(b, db); err != nil {
		log.Error("Failed to process pre-state")
		return err
	}

	//--------------------

	//epochMetaData := metadata.NewEpochMetaData(0, nil, nil)
	//s, err := state.NewStateContext(db, 0, b.ProtocolTransactions()[0].Pk, b.ParentHeaderHash(),
	//	b.HeaderHash(), b.PartialBlockSigningHash(), b.BlockSigningHash(), nil, epochMetaData)
	//
	//for _, pbData := range b.Transactions() {
	//	tx := transactions.ProtoToTransaction(pbData)
	//	switch tx.(type) {
	//	case *transactions.Transfer:
	//	case *transactions.Stake:
	//		if err := tx.ApplyEpochMetaData(epochMetaData); err != nil {
	//			log.Error("Failed to Apply EpochMetaData Changes")
	//			return err
	//		}
	//	}
	//}

	return b.Commit(db, b.HeaderHash(), true)
}
