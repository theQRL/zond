package commands

import (
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/wallet"
	"github.com/urfave/cli/v2"
)

func getDilithiumKeySubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "add",
			Usage: "Adds dilithium key from existing wallet",
			Flags: []cli.Flag{
				flags.WalletFile,
				flags.AccountIndexFlag,
				&cli.StringFlag{
					Name:  "output",
					Value: "dilithium_keys",
				},
			},
			Action: func(c *cli.Context) error {
				w := wallet.NewWallet(c.String(flags.WalletFile.Name))
				a, err := w.GetDilithiumAccountByIndex(c.Uint(flags.AccountIndexFlag.Name))
				if err != nil {
					return err
				}

				output := c.String("output")

				d := keys.NewDilithiumKeys(output)
				d.Add(a)
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List all dilithium keys",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "src",
					Value: "dilithium_keys",
				},
			},
			Action: func(c *cli.Context) error {
				w := keys.NewDilithiumKeys(c.String("src"))
				w.List()
				return nil
			},
		},
	}
}

func AddDilithiumKeyCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "dilithium-key",
		Usage: "Commands to manage Dilithium Keys",
		Flags: []cli.Flag{
			flags.WalletFile,
		},
		Subcommands: getDilithiumKeySubCommands(),
	})
}
