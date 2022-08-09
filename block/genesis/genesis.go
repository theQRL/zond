package genesis

import (
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/log"
	"github.com/theQRL/zond/protos"
	"io/ioutil"
	"path"
	"runtime"
)

func LoadPreState() (*protos.PreState, error) {
	_, filename, _, _ := runtime.Caller(0)
	directory := path.Dir(filename) // Current Package path
	preStateYAMLData, err := ioutil.ReadFile(path.Join(directory, path.Join("devnet", "prestate.yml")))

	if err != nil {
		log.Warn("Error while parsing prestate.yml")
		log.Info(err.Error())
		return nil, err
	}

	jsonData, err := yaml.YAMLToJSON(preStateYAMLData)
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
	_, filename, _, _ := runtime.Caller(0)
	directory := path.Dir(filename) // Current Package path
	genesisYAMLData, err := ioutil.ReadFile(path.Join(directory, path.Join("devnet", "genesis.yml")))

	if err != nil {
		log.Warn("Error while parsing Genesis.yml")
		log.Info(err.Error())
		return nil, err
	}

	jsonData, err := yaml.YAMLToJSON(genesisYAMLData)
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
