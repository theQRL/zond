package commands

import (
	"fmt"
	"github.com/theQRL/zond/block/genesis"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/transactions"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"path"
)

func getDevSubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "genesis-bootstrap",
			Usage: "Bootstraps the process of generating genesis file with required wallets and transactions",
			Action: func(c *cli.Context) error {
				if _, err := os.Stat("bootstrap"); !os.IsNotExist(err) {
					fmt.Println("bootstrap directory already exists, please delete it")
					return nil
				} else if err := os.Mkdir("bootstrap", os.ModePerm); err != nil {
					fmt.Println("failed to create bootstrap directory ", err.Error())
					return nil
				}

				dk := keys.NewDilithiumKeys(path.Join("bootstrap", "dilithium_keys"))

				walletFoundation := wallet.NewWallet(path.Join("bootstrap", "foundation-wallet.json"))
				walletFoundation.AddDilithium()
				foundationDilithiumAccount, err := walletFoundation.GetDilithiumAccountByIndex(1)
				if err != nil {
					fmt.Println("failed to get foundation dilithium account by index")
					return nil
				}
				dk.Add(foundationDilithiumAccount)

				targetWallets := config.GetDevConfig().BlocksPerEpoch * 2
				walletStake := wallet.NewWallet(path.Join("bootstrap", "wallet.json"))
				for i := uint(0); i < uint(targetWallets); i++ {
					walletStake.AddDilithium()
					stakeDilithiumAccount, err := walletStake.GetDilithiumAccountByIndex(i + 1)
					if err != nil {
						fmt.Println("failed to get dilithium account by index")
						return nil
					}
					dk.Add(stakeDilithiumAccount)
				}
				fmt.Println(fmt.Sprintf("Successfully generated %d dilithium address for staking", targetWallets))

				foundationDilithiumPK := foundationDilithiumAccount.GetPK()
				// nonce starts with 1 as foundation has signed coinbase transaction with nonce 0 for genesis block
				foundationAccountNonce := uint64(1)

				stakeAmount := 110000 * config.GetDevConfig().ShorPerQuanta
				gas := uint64(30000)
				gasPrice := uint64(10000)

				binAddress := foundationDilithiumAccount.GetAddress()
				address := misc.BytesToHexStr(binAddress[:])
				fmt.Println("Foundation Dilithium Address: ", address)

				tl := misc.NewTransactionList(path.Join("bootstrap", "genesisTransactions.json"))

				//Transactions to fund all validators from foundation dilithium account
				for i := uint(0); i < uint(targetWallets); i++ {
					stakeDilithiumAccount, err := walletStake.GetDilithiumAccountByIndex(i + 1)
					if err != nil {
						fmt.Println("failed to get stake dilithium account by index")
						return nil
					}

					binStakeDilithiumAddress := stakeDilithiumAccount.GetAddress()

					tx := transactions.NewTransfer(config.GetDevConfig().ChainID.Uint64(),
						binStakeDilithiumAddress[:],
						stakeAmount,
						gas,
						gasPrice,
						nil,
						foundationAccountNonce,
						foundationDilithiumPK[:])
					tx.SignDilithium(foundationDilithiumAccount, tx.GetSigningHash())
					tl.Add(tx.PBData())

					foundationAccountNonce += 1
				}

				// Adding foundation stake tx into the transaction list
				foundationStakeTx := transactions.NewStake(config.GetDevConfig().ChainID.Uint64(),
					config.GetDevConfig().StakeAmount,
					gas,
					gasPrice,
					foundationAccountNonce,
					foundationDilithiumPK[:])
				foundationStakeTx.SignDilithium(foundationDilithiumAccount, foundationStakeTx.GetSigningHash())
				foundationAccountNonce += 1
				tl.Add(foundationStakeTx.PBData())

				for i := uint(0); i < uint(targetWallets); i++ {
					stakeDilithiumAccount, err := walletStake.GetDilithiumAccountByIndex(i + 1)
					if err != nil {
						fmt.Println("failed to get stake dilithium account by index")
						return nil
					}

					stakeDilithiumPK := stakeDilithiumAccount.GetPK()

					tx := transactions.NewStake(config.GetDevConfig().ChainID.Uint64(),
						config.GetDevConfig().StakeAmount,
						gas,
						gasPrice,
						0,
						stakeDilithiumPK[:])
					tx.SignDilithium(stakeDilithiumAccount, tx.GetSigningHash())
					tl.Add(tx.PBData())
				}

				txs := tl.GetTransactions()

				genesisBlock, err := genesis.NewGenesisBlock(config.GetDevConfig().ChainID.Uint64(),
					txs, foundationDilithiumPK[:])

				genesisBlock.SignByProposer(foundationDilithiumAccount)

				preState := genesis.GeneratePreState(txs)

				preStateJSON, err := protojson.Marshal(preState)
				if err != nil {
					fmt.Println("Error: ", err.Error())
					return err
				}

				err = genesis.WriteYML(preStateJSON, path.Join("bootstrap", "prestate.yml"))
				if err != nil {
					return err
				}

				if err != nil {
					fmt.Println("failed to generate genesis and preStateFile")
					return err
				}

				genesisBlockJSON, err := protojson.Marshal(genesisBlock.PBData())
				if err != nil {
					fmt.Println("Error: ", err.Error())
					return err
				}

				return genesis.WriteYML(genesisBlockJSON, path.Join("bootstrap", "genesis.yml"))
			},
		},
	}
}

func AddDevCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:        "dev",
		Usage:       "Developer only command",
		Flags:       []cli.Flag{},
		Subcommands: getDevSubCommands(),
	})
}
