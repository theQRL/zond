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

func broadcastTransaction(transaction interface{}, url string, txHash []byte) error {
	responseBody := new(bytes.Buffer)
	err := json.NewEncoder(responseBody).Encode(transaction)
	if err != nil {
		fmt.Println("Error: ", err)
		return err
	}

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
	if responseData["transactionHash"].(string) == misc.Bin2HStr(txHash) {
		fmt.Println("Transaction successfully broadcasted")
	}
	return nil
}

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
				flags.TransactionFeeFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
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
				fee := c.Uint64(flags.TransactionFeeFlag.Name)
				output := c.String("output")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)

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
				tx := transactions.NewStake(c.Uint64(flags.NetworkIDFlag.Name), dilithiumPKs, true,
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

					url := fmt.Sprintf("http://%s/api/broadcast/stake", remoteAddr)
					return broadcastTransaction(stake, url, tx.TxHash(tx.GetSigningHash()))

				}
				return nil
			},
		},
		{
			Name: "transfer",
			Usage: "Generates a signed transfer transaction",
			Flags: []cli.Flag {
				flags.WalletFile,
				flags.XMSSIndexFlag,
				flags.NetworkIDFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag{
					Name: "address-to",
					Value: "",
				},
				&cli.Uint64Flag{
					Name: "amount",
					Value: 0,
				},
				flags.TransactionFeeFlag,
			},
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				xmss, err := w.GetXMSSByIndex(c.Uint(flags.XMSSIndexFlag.Name))
				if err != nil {
					return err
				}
				addressTo := c.String("address-to")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)
				binAddressTo := misc.HStr2Bin(addressTo)

				tx := transactions.NewTransfer(
					c.Uint64(flags.NetworkIDFlag.Name),
					[][]byte{binAddressTo},
					[]uint64{c.Uint64("amount")},
					c.Uint64("fee"),
					nil,
					nil,
					c.Uint64(flags.NonceFlag.Name),
					xmss.PK(),
					nil)
				tx.Sign(xmss, tx.GetSigningHash())

				if stdOut {
					jsonData, err := tx.ToJSON()
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}
					fmt.Println(misc.BytesToString(jsonData))
				}

				if broadcastFlag {
					transfer := view.PlainTransferTransaction{}
					transfer.TransactionFromPBData(tx.PBData(), tx.TxHash(tx.GetSigningHash()))

					url := fmt.Sprintf("http://%s/api/broadcast/transfer", remoteAddr)
					return broadcastTransaction(transfer, url, tx.TxHash(tx.GetSigningHash()))
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
