package genesis

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
)

func GeneratePreState(transactions []*protos.Transaction, outputFile string) error {
	preState := &protos.PreState{}

	preState.Transactions = transactions
	devConf := config.GetDevConfig()

	coinBaseAddressBalance := &protos.AddressBalance {
		Address: devConf.Genesis.CoinBaseAddress,
		Balance: devConf.Genesis.MaxCoinSupply - devConf.Genesis.SuppliedCoins,
	}

	xmssAddressBalance := &protos.AddressBalance {
		Address: devConf.Genesis.FoundationXMSSAddress,
		Balance: devConf.Genesis.SuppliedCoins,
	}

	preState.AddressBalance = append(preState.AddressBalance, coinBaseAddressBalance)
	preState.AddressBalance = append(preState.AddressBalance, xmssAddressBalance)

	jsonData, err := protojson.Marshal(preState)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	defer f.Close()
	f.Write(yamlData)

	f.Sync()

	return nil
}

func GenerateGenesis(networkID uint64, stakeTransactionsFile string,
	dilithiumGroup []*protos.DilithiumGroup, genesisFilename string,
	preStateFilename string) error {
	/*
	TODO: Select Network ID before generating genesis.yml
	This network id must be set in CoinBase Transaction and
	must be validated against transactions into the block.
	 */

	//if err := GeneratePreState(networkID, dilithiumGroup, xmss, preStateFilename); err != nil {
	//	return err
	//}

	c := config.GetDevConfig()

	tl := misc.NewTransactionList(stakeTransactionsFile)
	transactions := tl.GetTransactions()

	if err := GeneratePreState(transactions, preStateFilename); err != nil {
		return err
	}

	dilithiumInfos := dilithiumGroup[0].DilithiumInfo
	b := block.NewBlock(networkID, c.Genesis.GenesisTimestamp, misc.HStr2Bin(dilithiumInfos[0].PK),
		0,  c.Genesis.GenesisPrevHeaderHash, transactions, nil, 0)

	for i := 0; i < len(dilithiumGroup); i++ {
		for j := 0; j < len(dilithiumGroup[i].DilithiumInfo); j++ {
			if i == 0  && j == 0 {
				// Ignore as i == 0 and j == 0 is dilithiumInfo for the block proposer
				continue
			}
			dilithiumInfo := dilithiumGroup[i].DilithiumInfo[j]
			d := dilithium.RecoverDilithium(misc.HStr2Bin(dilithiumInfo.PK),
				misc.HStr2Bin(dilithiumInfo.SK))
			attestTx, err := b.Attest(networkID, d)
			if err != nil {
				return err
			}
			b.AddAttestTx(attestTx)
		}
	}
	d := dilithium.RecoverDilithium(misc.HStr2Bin(dilithiumInfos[0].PK),
		misc.HStr2Bin(dilithiumInfos[0].SK))
	b.SignByProposer(d)

	jsonData, err := protojson.Marshal(b.PBData())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	f, err := os.Create(genesisFilename)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	defer f.Close()
	f.Write(yamlData)

	f.Sync()

	return nil
}
