package genesis

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/theQRL/zond/chain/block"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/crypto/dilithium"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
)

func GeneratePreState(networkID uint64, dilithiumGroup []*protos.DilithiumGroup,
	xmss *crypto.XMSS, outputFile string) error {
	preState := &protos.PreState{}

	for _, group := range dilithiumGroup {
		var dilithiumPKs [][]byte
		for _, di := range group.DilithiumInfo {
			dilithiumPKs = append(dilithiumPKs, misc.HStr2Bin(di.PK))
		}
		// The signing of new stake transaction might not be required
		// TODO: Parameterize Fee and Nonce
		tx := transactions.NewStake(networkID, dilithiumPKs,
			true, 0, 1, xmss.PK(), nil)
		tx.Sign(xmss, tx.GetSigningHash())

		preState.Transactions = append(preState.Transactions, tx.PBData())
	}

	devConf := config.GetDevConfig()

	coinBaseAddressBalance := &protos.AddressBalance {
		Address: devConf.Genesis.CoinBaseAddress,
		Balance: devConf.Genesis.MaxCoinSupply - devConf.Genesis.SuppliedCoins,
	}

	xmssAddressBalance := &protos.AddressBalance {
		Address: xmss.Address(),
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
	dilithiumGroup []*protos.DilithiumGroup, xmss *crypto.XMSS,
	genesisFilename string, preStateFilename string) error {
	/*
	TODO: Select Network ID before generating genesis.yml
	This network id must be set in CoinBase Transaction and
	must be validated against transactions into the block.
	 */

	if err := GeneratePreState(networkID, dilithiumGroup, xmss, preStateFilename); err != nil {
		return err
	}

	c := config.GetDevConfig()

	//reader := bufio.NewReader(os.Stdin)
	//hexSeed, _ := reader.ReadString('\n')
	//binSeed := misc.HStr2Bin(hexSeed)
	//xmss := crypto.FromExtendedSeed(binSeed)

	tl := misc.NewTransactionList(stakeTransactionsFile)

	transactions := tl.GetTransactions()

	// TODO: Generate pre-state.yml
	dilithiumInfos := dilithiumGroup[0].DilithiumInfo
	b := block.NewBlock(networkID, misc.HStr2Bin(dilithiumInfos[0].PK), 0,
		c.Genesis.GenesisPrevHeaderHash, transactions, nil, 0)

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
