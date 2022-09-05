package genesis

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
)

func GeneratePreState(transactions []*protos.Transaction) *protos.PreState {
	preState := &protos.PreState{}

	preState.Transactions = transactions
	devConf := config.GetDevConfig()

	coinBaseAddressBalance := &protos.AddressBalance{
		Address: devConf.Genesis.CoinBaseAddress[:],
		Balance: devConf.Genesis.MaxCoinSupply - devConf.Genesis.SuppliedCoins,
	}

	//xmssAddressBalance := &protos.AddressBalance {
	//	Address: devConf.Genesis.FoundationXMSSAddress,
	//	Balance: devConf.Genesis.SuppliedCoins,
	//}

	preState.AddressBalance = append(preState.AddressBalance, coinBaseAddressBalance)
	//preState.AddressBalance = append(preState.AddressBalance, xmssAddressBalance)

	return preState
}

func NewGenesisBlock(chainID uint64, txs []*protos.Transaction,
	foundationDilithiumPK []byte) (*block.Block, error) {

	c := config.GetDevConfig()

	b := block.NewBlock(chainID, c.Genesis.GenesisTimestamp, foundationDilithiumPK,
		0, c.Genesis.GenesisPrevHeaderHash, txs, nil, 0)
	return b, nil
}

func GenerateGenesis(chainID uint64, stakeTransactionsFile string,
	dilithiumInfos []*protos.DilithiumInfo, genesisFilename string,
	preStateFilename string) error {
	/*
		TODO: Select Network ID before generating genesis.yml
		This network id must be set in CoinBase Transaction and
		must be validated against transactions into the block.
	*/

	//if err := GeneratePreState(chainID, dilithiumGroup, xmss, preStateFilename); err != nil {
	//	return err
	//}

	c := config.GetDevConfig()

	tl := misc.NewTransactionList(stakeTransactionsFile)
	transactions := tl.GetTransactions()

	preState := GeneratePreState(transactions)
	preStateJSON, err := protojson.Marshal(preState)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	err = WriteYML(preStateJSON, preStateFilename)
	if err != nil {
		return err
	}

	binDilithiumPK, err := misc.HexStrToBytes(dilithiumInfos[0].PK)
	if err != nil {
		return err
	}

	b := block.NewBlock(chainID, c.Genesis.GenesisTimestamp, binDilithiumPK,
		0, c.Genesis.GenesisPrevHeaderHash, transactions, nil, 0)

	//for i := 0; i < len(dilithiumGroup); i++ {
	//	for j := 0; j < len(dilithiumGroup[i].DilithiumInfo); j++ {
	//		if i == 0  && j == 0 {
	//			// Ignore as i == 0 and j == 0 is dilithiumInfo for the block proposer
	//			continue
	//		}
	//		dilithiumInfo := dilithiumGroup[i].DilithiumInfo[j]
	//		binDilithiumPK, err := misc.HexStrToBytes(dilithiumInfo.PK)
	//		if err != nil {
	//			return err
	//		}
	//		binDilithiumSK, err := misc.HexStrToBytes(dilithiumInfo.SK)
	//		if err != nil {
	//			return err
	//		}
	//		d := dilithium.RecoverDilithium(binDilithiumPK, binDilithiumSK)
	//		attestTx, err := b.Attest(chainID, d)
	//		if err != nil {
	//			return err
	//		}
	//		b.AddAttestTx(attestTx)
	//	}
	//}
	//binDilithiumPK, err := misc.HexStrToBytes(dilithiumInfos[0].PK)
	//if err != nil {
	//	return err
	//}
	//binDilithiumSK, err := misc.HexStrToBytes(dilithiumInfos[0].SK)
	//if err != nil {
	//	return err
	//}
	//d := dilithium.RecoverDilithium(binDilithiumPK, binDilithiumSK)
	//b.SignByProposer(d)

	genesisBlockJSON, err := protojson.Marshal(b.PBData())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	err = WriteYML(genesisBlockJSON, genesisFilename)
	if err != nil {
		return err
	}

	return nil
}

func WriteYML(jsonData []byte, filename string) error {
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	defer f.Close()

	_, err = f.Write(yamlData)
	if err != nil {
		return err
	}

	return f.Sync()
}
