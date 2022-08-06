package genesis

import (
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/block/genesis/devnet"
	"github.com/theQRL/zond/protos"
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

func GenesisBlock() (*block.Block, error) {
	jsonData, err := yaml.YAMLToJSON(devnet.DevnetGenesisData["genesis.yml"])
	if err != nil {
		return nil, err
	}

	pbData := &protos.Block{}
	err = jsonpb.UnmarshalString(string(jsonData), pbData)
	if err != nil {
		return nil, err
	}

	b := block.BlockFromPBData(pbData)
	return b, nil
}
