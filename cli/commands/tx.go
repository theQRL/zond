package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/api/view"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto"
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

	if response.Error != 0 {
		fmt.Println(response.ErrorMessage)
	} else {
		responseData := response.Data.(map[string]interface{})
		if responseData["transactionHash"].(string) == misc.BytesToHexStr(txHash[:]) {
			fmt.Println("Transaction successfully broadcasted")
		}
	}
	return nil
}

type (
	RPCMessage struct {
		Version string          `json:"jsonrpc,omitempty"`
		ID      json.RawMessage `json:"id,omitempty"`
		Method  string          `json:"method,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
		Error   *JsonError      `json:"error,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
	}
	JsonError struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	}
)

func broadcastTransactionViaRPC(transaction interface{}, url string, txHash common.Hash) error {
	params := []interface{}{transaction}
	p, _ := json.Marshal(params)
	id, _ := json.Marshal(1)
	reqBody := RPCMessage{
		Version: "2.0",
		ID:      id,
		Method:  "zond_sendRawTransaction",
		Params:  p,
	}
	responseBody := new(bytes.Buffer)
	err := json.NewEncoder(responseBody).Encode(reqBody)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	req, err := http.NewRequest(http.MethodPost, url, responseBody)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var response RPCMessage
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if response.Error != nil {
		fmt.Println(response.Error)
	} else {
		hashed := ""
		err := json.Unmarshal(response.Result, &hashed)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if hashed == misc.BytesToHexStr(txHash[:]) {
			fmt.Println("Transaction successfully broadcasted")
		}
	}
	return nil
}

func evmCall(contractAddress string, data string, url string) (string, error) {
	evmCallReq := view.EVMCall{Address: contractAddress, Data: data}
	responseBody := new(bytes.Buffer)
	err := json.NewEncoder(responseBody).Encode(evmCallReq)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, responseBody)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	var response api.Response
	err = json.Unmarshal(bodyBytes, &response)

	fmt.Println(response.Data)
	responseData := response.Data.(map[string]interface{})
	return responseData["result"].(string), nil
}

func getTransactionSubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "stake",
			Usage: "Generates a signed stake transaction using Dilithium account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.ChainIDFlag,
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
				flags.RemoteRPCAddrFlag,
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
				stakeAmount := c.Uint64(flags.AmountFlag.Name)
				minStakeAmount := config.GetDevConfig().StakeAmount
				if stakeAmount != 0 && stakeAmount < minStakeAmount {
					fmt.Println(fmt.Sprintf("stake amount must be greater than or equals to %d QRL",
						minStakeAmount/config.GetDevConfig().ShorPerQuanta))
					fmt.Println("In case you are trying to de-stake use stakeAmount 0")
					return nil
				}

				output := c.String("output")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteRPCAddrFlag.Name)

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				dilithiumKeys := keys.NewDilithiumKeys(dilithiumFile)
				if stakeAmount >= minStakeAmount {
					dilithiumKeys.Add(a)
				} else {
					dilithiumKeys.Remove(a)
				}

				pk := a.GetPK()
				tx := transactions.NewStake(c.Uint64(flags.ChainIDFlag.Name),
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
					stake := view.PlainStakeTransactionRPC{}
					stake.TransactionFromPBData(tx.PBData())

					url := fmt.Sprintf("%s", remoteAddr)
					return broadcastTransactionViaRPC(stake, url, txHash)

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
				flags.ChainIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				&cli.StringFlag{
					Name:  "to",
					Value: "",
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := misc.HexStrToBytes(c.String(flags.DataFlag.Name))
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

				addressTo := c.String("to")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteRPCAddrFlag.Name)
				binAddressTo, err := misc.HexStrToBytes(addressTo)
				if err != nil {
					return err
				}

				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.ChainIDFlag.Name),
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
					transfer := view.PlainTransferTransactionRPC{}
					transfer.TransactionFromPBData(tx.PBData())

					url := fmt.Sprintf("%s", remoteAddr)
					return broadcastTransactionViaRPC(transfer, url, txHash)
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
				flags.ChainIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				flags.RemoteRPCAddrFlag,
				&cli.StringFlag{
					Name:  "to",
					Value: "",
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := misc.HexStrToBytes(c.String(flags.DataFlag.Name))
				if err != nil {
					fmt.Println("error decoding data")
					return err
				}

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				addressTo := c.String("to")
				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteRPCAddrFlag.Name)
				binAddressTo, err := misc.HexStrToBytes(addressTo)
				if err != nil {
					return err
				}

				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.ChainIDFlag.Name),
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
					transfer := view.PlainTransferTransactionRPC{}
					transfer.TransactionFromPBData(tx.PBData())

					url := fmt.Sprintf("%s", remoteAddr)
					return broadcastTransactionViaRPC(transfer, url, txHash)
				}
				return nil
			},
		},
		{
			Name:  "deployContractFromDilithium",
			Usage: "Deploys a smart contract using Dilithium account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.ChainIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				flags.RemoteRPCAddrFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := misc.HexStrToBytes(c.String(flags.DataFlag.Name))
				if err != nil {
					fmt.Println("error decoding data")
					return err
				}

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteRPCAddrFlag.Name)
				nonce := c.Uint64(flags.NonceFlag.Name)

				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.ChainIDFlag.Name),
					nil,
					c.Uint64(flags.AmountFlag.Name),
					c.Uint64(flags.GasFlag.Name),
					c.Uint64(flags.GasPriceFlag.Name),
					data,
					nonce,
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

				contractAddr := crypto.CreateAddress(a.GetAddress(), nonce)
				fmt.Println("Contract Address: ", contractAddr)

				if broadcastFlag {
					txHash := tx.Hash()
					transfer := view.PlainTransferTransactionRPC{}
					transfer.TransactionFromPBData(tx.PBData())

					url := fmt.Sprintf("%s", remoteAddr)
					return broadcastTransactionViaRPC(transfer, url, txHash)
				}
				return nil
			},
		},
		{
			Name:  "callContractFromDilithium",
			Usage: "Calls contract using Dilithium account",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				flags.ChainIDFlag,
				flags.DataFlag,
				flags.NonceFlag,
				&cli.StringFlag{
					Name:     "to",
					Value:    "",
					Required: true,
				},
				flags.AmountFlag,
				flags.GasFlag,
				flags.GasPriceFlag,
				flags.TransactionStdOut,
				flags.BroadcastFlag,
				flags.RemoteAddrFlag,
				flags.RemoteRPCAddrFlag,
			},
			Action: func(c *cli.Context) error {
				data, err := misc.HexStrToBytes(c.String(flags.DataFlag.Name))
				if err != nil {
					fmt.Println("error decoding data")
					return err
				}

				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				stdOut := c.Bool(flags.TransactionStdOut.Name)
				broadcastFlag := c.Bool(flags.BroadcastFlag.Name)
				remoteAddr := c.String(flags.RemoteRPCAddrFlag.Name)
				to := c.String("to")
				nonce := c.Uint64(flags.NonceFlag.Name)

				binTo, err := misc.HexStrToBytes(to)
				if err != nil {
					fmt.Println("failed to decode to address")
					return err
				}
				pk := a.GetPK()
				tx := transactions.NewTransfer(
					c.Uint64(flags.ChainIDFlag.Name),
					binTo,
					c.Uint64(flags.AmountFlag.Name),
					c.Uint64(flags.GasFlag.Name),
					c.Uint64(flags.GasPriceFlag.Name),
					data,
					nonce,
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
					transfer := view.PlainTransferTransactionRPC{}
					transfer.TransactionFromPBData(tx.PBData())

					url := fmt.Sprintf("%s", remoteAddr)
					return broadcastTransactionViaRPC(transfer, url, txHash)
				}
				return nil
			},
		},
		{
			Name:  "offChainCallContract",
			Usage: "Off-chain contract call",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "contract-address",
					Value:    "",
					Required: true,
				},
				flags.DataFlag,
				flags.RemoteAddrFlag,
			},
			Action: func(c *cli.Context) error {
				data := c.String(flags.DataFlag.Name)
				remoteAddr := c.String(flags.RemoteAddrFlag.Name)

				contractAddress := c.String("contract-address")

				url := fmt.Sprintf("http://%s/api/evmcall", remoteAddr)
				result, err := evmCall(contractAddress, data, url)
				if err != nil {
					fmt.Println("error while making evmCall")
					return err
				}
				fmt.Println("Result: ", result)
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
