package flags

import (
	"fmt"
	"github.com/theQRL/zond/config"
	"github.com/urfave/cli/v2"
)

// List of global flags used by CLI commands

var WalletFile = &cli.StringFlag {
	Name: "wallet-file",
	Value: "wallet.json",  // TODO: Move this to Dev Config
}

var XMSSIndexFlag = &cli.UintFlag {
	Name: "xmss-index",
	Value: 1,
	Required: true,
}

var OTSKeyIndexFlag = &cli.UintFlag {
	Name: "ots-key-index",
	Value: 0,
	Required: true,
}

var NetworkIDFlag = &cli.Uint64Flag {
	Name: "network-id",
	Value: 1,
	Required: true,
}

var NonceFlag = &cli.Uint64Flag {
	Name: "nonce",
	Value: 1,
	Required: true,
}

var BroadcastFlag = &cli.BoolFlag {
	Name: "broadcast",
	Value: false,
	Required: false,
}

var RemoteAddrFlag = &cli.StringFlag {
	Name: "remote-addr",
	Value: fmt.Sprintf("%s:%d",
		config.GetUserConfig().API.PublicAPI.Host,
		config.GetUserConfig().API.PublicAPI.Port),
	Required: false,
}

var TransactionFeeFlag = &cli.Uint64Flag {
	Name: "fee",
	Value: 0,
}

var TransactionStdOut = &cli.BoolFlag {
	Name: "std-out",
	Value: true,
}