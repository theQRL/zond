package flags

import "github.com/urfave/cli/v2"

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