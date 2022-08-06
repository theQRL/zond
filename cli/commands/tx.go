package commands

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/api/view"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/transactions"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
)

func broadcastTransaction(transaction interface{}, url string, txHash common.Hash) error {
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
	if responseData["transactionHash"].(string) == hex.EncodeToString(txHash[:]) {
		fmt.Println("Transaction successfully broadcasted")
	}
	return nil
}

func getTransactionSubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "stake",
			Usage: "Generates a signed stake transaction using Dilithium account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.NetworkIDFlag,
				&cli.StringFlag{
					Name:  "dilithium-file",
					Value: "dilithium_keys",
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag{
					Name:  "output",
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
				stakeAmount := c.Uint64(flags.AmountFlag.Name) * config.GetDevConfig().ShorPerQuanta
				minStakeAmount := config.GetDevConfig().StakeAmount
				if stakeAmount != 0 && stakeAmount < minStakeAmount {
					fmt.Println("stake amount must be greater than or equals to ",
						minStakeAmount/config.GetDevConfig().ShorPerQuanta)
				}

				output := c.String("output")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				dilithiumKeys := keys.NewDilithiumKeys(dilithiumFile)
				dilithiumKeys.Add(a)

				pk := a.GetPK()
				tx := transactions.NewStake(c.Uint64(flags.NetworkIDFlag.Name),
					stakeAmount,
					c.Uint64(flags.GasFlag.Name),
					c.Uint64(flags.GasPriceFlag.Name),
					c.Uint64(flags.NonceFlag.Name),
					pk[:])
				tx.SignDilithium(a, tx.GetSigningHash())

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
					txHash := tx.Hash()
					stake := view.PlainStakeTransaction{}
					stake.TransactionFromPBData(tx.PBData(), txHash[:])

					url := fmt.Sprintf("http://%s/api/broadcast/stake", remoteAddr)
					return broadcastTransaction(stake, url, txHash)

				}
				return nil
			},
		},
		{
			Name:  "transferFromXMSS",
			Usage: "Generates a signed transfer transaction using XMSS account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.OTSKeyIndexFlag,
				flags.NetworkIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag{
					Name:  "address-to",
					Value: "",
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := hex.DecodeString(flags.DataFlag.Name)
				if err != nil {
					fmt.Println("error decoding data")
					return err
				}

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetXMSSAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}
				a.SetIndex(uint32(c.Uint(flags.OTSKeyIndexFlag.Name)))

				addressTo := c.String("address-to")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)
				binAddressTo, err := hex.DecodeString(addressTo)
				if err != nil {
					return err
				}

				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.NetworkIDFlag.Name),
					binAddressTo,
					c.Uint64(flags.AmountFlag.Name),
					c.Uint64(flags.GasFlag.Name),
					c.Uint64(flags.GasPriceFlag.Name),
					data,
					c.Uint64(flags.NonceFlag.Name),
					pk[:])
				tx.SignXMSS(a, tx.GetSigningHash())

				if stdOut {
					jsonData, err := tx.ToJSON()
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}
					fmt.Println(misc.BytesToString(jsonData))
				}

				if broadcastFlag {
					txHash := tx.Hash()
					transfer := view.PlainTransferTransaction{}
					transfer.TransactionFromPBData(tx.PBData(), txHash[:])

					url := fmt.Sprintf("http://%s/api/broadcast/transfer", remoteAddr)
					return broadcastTransaction(transfer, url, txHash)
				}
				return nil
			},
		},
		{
			Name:  "transferFromDilithium",
			Usage: "Generates a signed transfer transaction using Dilithium account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.NetworkIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag{
					Name:  "address-to",
					Value: "",
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := hex.DecodeString(flags.DataFlag.Name)
				if err != nil {
					fmt.Println("error decoding data")
					return err
				}

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				addressTo := c.String("address-to")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)
				binAddressTo, err := hex.DecodeString(addressTo)
				if err != nil {
					return err
				}

				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.NetworkIDFlag.Name),
					binAddressTo,
					c.Uint64(flags.AmountFlag.Name),
					c.Uint64(flags.GasFlag.Name),
					c.Uint64(flags.GasPriceFlag.Name),
					data,
					c.Uint64(flags.NonceFlag.Name),
					pk[:])
				tx.SignDilithium(a, tx.GetSigningHash())

				if stdOut {
					jsonData, err := tx.ToJSON()
					if err != nil {
						fmt.Println("Error: ", err)
						return err
					}
					fmt.Println(misc.BytesToString(jsonData))
				}

				if broadcastFlag {
					txHash := tx.Hash()
					transfer := view.PlainTransferTransaction{}
					transfer.TransactionFromPBData(tx.PBData(), txHash[:])

					url := fmt.Sprintf("http://%s/api/broadcast/transfer", remoteAddr)
					return broadcastTransaction(transfer, url, txHash)
				}
				return nil
			},
		},
	}
}

func AddTransactionCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:        "tx",
		Usage:       "Commands to generate tx",
		Subcommands: getTransactionSubCommands(),
	})
}
