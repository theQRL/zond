package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/api/view"
	"github.com/theQRL/zond/chain/transactions"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
)

func getTransactionSubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name: "stake",
			Usage: "Generates a signed stake transaction",
			Flags: []cli.Flag {
				flags.WalletFile,
				flags.XMSSIndexFlag,
				flags.NetworkIDFlag,
				&cli.StringFlag {
					Name: "dilithium-file",
					Value: "dilithium_keys",
				},
				&cli.UintFlag {
					Name: "dilithium-group-index",
					Value: 1,
					Required: true,
				},
				&cli.Uint64Flag {
					Name: "fee",
					Value: 0,  // TODO: Value must be derived from config
				},
				flags.NonceFlag,
				&cli.BoolFlag {
					Name: "std-out",
					Value: true,
				},
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag {
					Name: "output",
					Value: "stake_transactions.json",
				},
			},
			Action: func(c *cli.Context) error {
				/*
					1. Load the Wallet file with XMSS address based on index
					2. Read file dilithium key
					3. Pick the group based on group index
					4. Load the Dilithium Keys from the group and form the transaction
					5. Form the Stake transaction and sign it by XMSS
				*/
				dilithiumFile := c.String("dilithium-file")
				dilithiumGroupIndex := c.Uint("dilithium-group-index")
				fee := c.Uint64("fee")
				output := c.String("output")
				stdOut := c.Bool("std-out")
				broadcastFlag := c.Bool("broadcast")
				remoteAddr := c.String("remote-addr")

				w := wallet.NewWallet(c.String("wallet-file"))
				xmss, err := w.GetXMSSByIndex(c.Uint("xmss-index"))
				if err != nil {
					return err
				}
				dilithiumKeys := keys.NewDilithiumKeys(dilithiumFile)
				dilithiumGroup, err := dilithiumKeys.GetDilithiumGroupByIndex(dilithiumGroupIndex)
				if err != nil {
					return err
				}
				var dilithiumPKs [][]byte
				for _, dilithiumInfo := range dilithiumGroup.DilithiumInfo {
					dilithiumPKs = append(dilithiumPKs, misc.HStr2Bin(dilithiumInfo.PK))
				}
				tx := transactions.NewStake(c.Uint64("network-id"), dilithiumPKs, true,
					fee, c.Uint64("nonce"), xmss.PK(), nil)
				tx.Sign(xmss, tx.GetSigningHash())

				if len(output) > 0 {
					tl := misc.NewTransactionList(output)
					tl.Add(tx.PBData())
					tl.Save()
				}

				if stdOut {
					jsonData, err := tx.ToJSON()
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}
					fmt.Println(misc.BytesToString(jsonData))
				}

				if broadcastFlag {
					stake := view.PlainStakeTransaction{}
					stake.TransactionFromPBData(tx.PBData(), tx.TxHash(tx.GetSigningHash()))

					responseBody := new(bytes.Buffer)
					err := json.NewEncoder(responseBody).Encode(stake)
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}

					url := fmt.Sprintf("http://%s/api/broadcast/stake", remoteAddr)
					req, err := http.NewRequest(http.MethodPost, url, responseBody)
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					bodyBytes, _ := ioutil.ReadAll(resp.Body)

					var response api.Response
					err = json.Unmarshal(bodyBytes, &response)

					responseData := response.Data.(map[string]interface{})
					if responseData["transactionHash"].(string) == misc.Bin2HStr(tx.TxHash(tx.GetSigningHash())) {
						fmt.Println("Transaction successfully broadcasted")
					}
				}
				return nil
			},
		},
	}
}

func AddTransactionCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name: "tx",
		Usage: "Commands to generate tx",
		Subcommands: getTransactionSubCommands(),
	})
}
