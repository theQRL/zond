package main

import (
	"fmt"
	"github.com/theQRL/zond/cli/commands"
	"github.com/urfave/cli/v2"
	"os"
)

var app = cli.NewApp()

func info() {
	app.Name = "Zond CLI"
	app.Usage = "Zond CLI"
	app.Version = "0.0.1"
}

func initCommands() {
	app.Commands = []*cli.Command{}
	commands.AddWalletCommand(app)
	commands.AddDilithiumKeyCommand(app)
	commands.AddTransactionCommand(app)
	commands.AddGenesisCommand(app)
	commands.AddDevCommand(app)
}

func main() {
	info()
	initCommands()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
