package commands

import (
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
)

func getWalletSubCommands() []*cli.Command {
	heightFlag := &cli.Uint64Flag{
		Name: "height",
		Value: 10,  // TODO: Move this to Dev Config
	}
	return []*cli.Command {
		{
			Name: "add",
			Usage: "Adds a new XMSS address into the Wallet",
			Flags: []cli.Flag {
				heightFlag,
			},
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(flags.WalletFile.Value)
				w.Add(heightFlag.Value, goqrllib.SHA2_256)

				fmt.Println("Wallet Created")
				return nil
			},
		},
		{
			Name: "list",
			Usage: "List all addresses in a Wallet",
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(flags.WalletFile.Value)
				w.List()
				return nil
			},
		},
	}
}

func AddWalletCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name: "wallet",
		Usage: "Commands to manage Zond Wallet",
		Flags: []cli.Flag {
			flags.WalletFile,
		},
		Subcommands: getWalletSubCommands(),
	})
}
