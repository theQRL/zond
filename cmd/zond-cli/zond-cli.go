package main

import (
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli"
	"os"
)

var app = cli.NewApp()

var walletFile = &cli.StringFlag {
	Name: "wallet-file",
	Value: "wallet.json",  // TODO: Move this to Dev Config
}

func info() {
	app.Name = "Zond CLI"
	app.Usage = "Zond CLI"
	app.Version = "0.0.1"
}

func getWalletCommands() []*cli.Command {
	heightFlag := &cli.Uint64Flag{
		Name: "height",
		Value: 10,  // TODO: Move this to Dev Config
	}
	return []*cli.Command {
		{
			Name: "add",
			Usage: "Add a new XMSS address into the Wallet",
			Flags: []cli.Flag {
				heightFlag,
			},
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(walletFile.Value)
				w.Add(heightFlag.Value, goqrllib.SHA2_256)

				fmt.Println("Wallet Created")
				return nil
			},
		},
		{
			Name: "list",
			Usage: "List all addresses in a Wallet",
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(walletFile.Value)
				w.List()
				return nil
			},
		},
	}
}

func commands() {
	app.Commands = []*cli.Command{
		{
			Name: "wallet",
			Usage: "Zond Wallet",
			Flags: []cli.Flag {
				walletFile,
			},
			Subcommands: getWalletCommands(),
		},
	}
}

func main() {
	info()
	commands()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
