package flags

import (
	"fmt"
	"github.com/theQRL/zond/config"
	"github.com/urfave/cli/v2"
)

// List of global flags used by CLI commands

var WalletFile = &cli.StringFlag{
	Name:  "wallet-file",
	Value: "wallet.json", // TODO: Move this to Dev Config
}

var AccountIndexFlag = &cli.UintFlag{
	Name:     "account-index",
	Value:    1,
	Required: true,
}

var OTSKeyIndexFlag = &cli.UintFlag{
	Name:     "ots-key-index",
	Value:    0,
	Required: true,
}

var ChainIDFlag = &cli.Uint64Flag{
	Name:     "chain-id",
	Value:    1,
	Required: false,
}

var DataFlag = &cli.StringFlag{
	Name:     "data",
	Value:    "",
	Required: false,
}

var NonceFlag = &cli.Uint64Flag{
	Name:     "nonce",
	Value:    1,
	Required: true,
}

var BroadcastFlag = &cli.BoolFlag{
	Name:     "broadcast",
	Value:    false,
	Required: false,
}

var RemoteAddrFlag = &cli.StringFlag{
	Name: "remote-addr",
	Value: fmt.Sprintf("%s:%d",
		config.GetUserConfig().API.PublicAPI.Host,
		config.GetUserConfig().API.PublicAPI.Port),
	Required: false,
}

var RemoteRPCAddrFlag = &cli.StringFlag{
	Name: "remote-rpc-addr",
	Value: fmt.Sprintf("http://%s:%d",
		config.GetUserConfig().API.PublicAPIRpc.Host,
		config.GetUserConfig().API.PublicAPIRpc.Port),
	Required: false,
}

var AmountFlag = &cli.Uint64Flag{
	Name:     "amount",
	Value:    0,
	Required: true,
}

var GasFlag = &cli.Uint64Flag{
	Name:     "gas",
	Value:    0,
	Required: true,
}

var GasPriceFlag = &cli.Uint64Flag{
	Name:     "gas-price",
	Value:    0,
	Required: true,
}

var TransactionStdOut = &cli.BoolFlag{
	Name:  "std-out",
	Value: true,
}
