package commands

import (
	"fmt"
	"github.com/theQRL/zond/cli/flags"
	"github.com/theQRL/zond/keys"
	"github.com/theQRL/zond/protos"
	"github.com/urfave/cli/v2"
)

func getDilithiumKeySubCommands() []*cli.Command {
	return []*cli.Command {
		{
			Name: "add",
			Usage: "Adds new dilithium key pair(s)",
			Flags: []cli.Flag {
				&cli.IntFlag {
					Name: "count",
					Value: 1,
				},
				&cli.StringFlag {
					Name: "output",
					Value: "dilithium_keys",
				},
			},
			Action: func(c *cli.Context) error {
				count := c.Value("count").(int)
				output := c.String("output")
				if count > 100 || count < 1 {
					fmt.Println("Error: Count value cannot be more than 100 or less than 1")
					return nil
				}
				d := keys.NewDilithiumKeys(output)
				dilithiumGroup := &protos.DilithiumGroup{}
				for i := 0; i < count; i++ {
					d.Add(dilithiumGroup)
				}
				d.AddGroup(dilithiumGroup)
				return nil
			},
		},
		{
			Name: "list",
			Usage: "List all dilithium keys",
			Flags: []cli.Flag {
				&cli.StringFlag {
					Name: "src",
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
		Name: "dilithium-key",
		Usage: "Commands to manage Dilithium Keys",
		Flags: []cli.Flag {
			flags.WalletFile,
		},
		Subcommands: getDilithiumKeySubCommands(),
	})
}
