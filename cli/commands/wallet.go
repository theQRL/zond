package commands

import (
	"fmt"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
)

func getWalletSubCommands() []*cli.Command {
	heightFlag := &cli.Uint64Flag{
		Name:  "height",
		Value: 10, // TODO: Move this to Dev Config
	}
	return []*cli.Command{
		{
			Name:  "add-xmss",
			Usage: "Adds a new XMSS address into the Wallet",
			Flags: []cli.Flag{
				heightFlag,
			},
			Action: func(c *cli.Context) error {
				hashType := xmss.SHA2_256
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				w.AddXMSS(uint8(heightFlag.Value), hashType)

				fmt.Println("Wallet Created")
				return nil
			},
		},
		{
			Name:  "add-dilithium",
			Usage: "Adds a new Dilithium address into the Wallet",
			Flags: []cli.Flag{},
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				w.AddDilithium()

				fmt.Println("Wallet Created")
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List all addresses in a Wallet",
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				w.List()
				return nil
			},
		},
		{
			Name:  "secret",
			Usage: "Show hexseed & mnemonic for the addresses in Wallet",
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				w.Secret()
				return nil
			},
		},
	}
}

func AddWalletCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "wallet",
		Usage: "Commands to manage Zond Wallet",
		Flags: []cli.Flag{
			flags.WalletFile,
		},
		Subcommands: getWalletSubCommands(),
	})
}
