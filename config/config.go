package config

import (
	"github.com/theQRL/zond/misc"
	"os/user"
	"path"
	"sync"
)

type Config struct {
	Dev  *DevConfig
	User *UserConfig
}

type NodeConfig struct {
	EnablePeerDiscovery      bool
	PeerList                 []string
	BindingIP                string
	LocalPort                uint16
	PublicPort               uint16
	PeerRateLimit            uint64
	BanMinutes               uint8
	MaxPeersLimit            uint16
	MaxPeersInPeerList       uint64
	MaxRedundantConnections  int
}

type NTPConfig struct {
	Retries int
	Servers []string
	Refresh uint64
}

type TransactionPoolConfig struct {
	TransactionPoolSize           uint64
	PendingTransactionPoolSize    uint64
	PendingTransactionPoolReserve uint64
	StaleTransactionThreshold     uint64
}

type StakeConfig struct {
	EnableStaking bool
	DilithiumKeysFileName string
}

//type API struct {
//	AdminAPI  *APIConfig
//	PublicAPI *APIConfig
//}

//type MongoProcessorConfig struct {
//	Enabled  bool
//	DBName   string
//	Host     string
//	Port     uint16
//	Username string
//	Password string
//
//	ItemsPerPage uint64
//}

type UserConfig struct {
	Node *NodeConfig

	NTP *NTPConfig

	ChainStateTimeout         uint16
	ChainStateBroadcastPeriod uint16

	TransactionPool *TransactionPoolConfig
	Stake           *StakeConfig

	BaseDir            string
	ChainFileDirectory string

	//	API                  *API
	//	MongoProcessorConfig *MongoProcessorConfig
}

//type APIConfig struct {
//	Enabled          bool
//	Host             string
//	Port             uint32
//	Threads          uint32
//	MaxConcurrentRPC uint16
//}

type DevConfig struct {
	Genesis *GenesisConfig

	Version string

	BlocksPerEpoch       uint64
	BlockLeadTimestamp   uint32
	BlockMaxDrift        uint16
	MaxFutureBlockLength uint16
	MaxMarginBlockNumber uint16
	MinMarginBlockNumber uint16

	ReorgLimit uint64

	MessageQSize          uint32
	MessageReceiptTimeout uint32
	MessageBufferSize     uint32

	OTSBitFieldPerPage uint64

	DefaultNonce          uint8
	DefaultAccountBalance uint64
	BlockTime             uint64

	DBName              string
	PeersFilename       string
	WalletDatFilename   string
	BannedPeersFilename string

	Transaction *TransactionConfig

	NumberOfBlockAnalyze uint8
	SizeMultiplier       float64
	BlockMinSizeLimit    int
	TxExtraOverhead      int

	ShorPerQuanta uint64

	MaxReceivableBytes uint64
	ReservedQuota      uint64
	MaxBytesOut        uint64

	BlockTimeSeriesSize uint32

	RecordTransactionHashes bool // True will enable recording of transaction hashes into address state
}

type TransactionConfig struct {
	MultiOutputLimit uint8
}

type GenesisConfig struct {
	GenesisPrevHeaderHash []byte
	MaxCoinSupply         uint64
	SuppliedCoins         uint64
	GenesisDifficulty     uint64
	CoinBaseAddress       []byte
	GenesisTimestamp      uint64
}

var once sync.Once
var config *Config

func GetConfig() *Config {
	once.Do(func() {
		userConfig := GetUserConfig()
		devConfig := GetDevConfig()
		config = &Config{
			User: userConfig,
			Dev:  devConfig,
		}
	})

	return config
}

func GetUserConfig() (userConf *UserConfig) {
	node := &NodeConfig{
		EnablePeerDiscovery: true,
		PeerList: []string{
		},
		BindingIP:               "0.0.0.0",
		LocalPort:               15005,
		PublicPort:              15005,
		PeerRateLimit:           500,
		BanMinutes:              20,
		MaxPeersLimit:           32,
		MaxPeersInPeerList:      100,
		MaxRedundantConnections: 5,
	}

	ntp := &NTPConfig{
		Retries: 6,
		Servers: []string{"pool.ntp.org", "ntp.ubuntu.com"},
		Refresh: 12 * 60 * 60,
	}

	transactionPool := &TransactionPoolConfig{
		TransactionPoolSize:           25000,
		PendingTransactionPoolSize:    75000,
		PendingTransactionPoolReserve: 750,
		StaleTransactionThreshold:     15,
	}

	//	adminAPI := &APIConfig{
	//		Enabled:          false,
	//		Host:             "127.0.0.1",
	//		Port:             19008,
	//		Threads:          1,
	//		MaxConcurrentRPC: 100,
	//	}
	//
	//	publicAPI := &APIConfig{
	//		Enabled:          true,
	//		Host:             "127.0.0.1",
	//		Port:             19009,
	//		Threads:          1,
	//		MaxConcurrentRPC: 100,
	//	}
	//
	//
	//	api := &API{
	//		AdminAPI:  adminAPI,
	//		PublicAPI: publicAPI,
	//	}
	//	mongoProcessorConfig := &MongoProcessorConfig{
	//		Enabled:      false,
	//		DBName:       "zond",
	//		Host:         "127.0.0.1",
	//		Port:         3001,
	//		Username:     "",
	//		Password:     "",
	//		ItemsPerPage: 1000,
	//	}
	userCurrentDir, _ := user.Current() // TODO: Handle error
	stake := &StakeConfig{
		EnableStaking: false,
		DilithiumKeysFileName: path.Join(path.Join(userCurrentDir.HomeDir,
			".zond"), "dilithium_keys"),
	}
	userConf = &UserConfig{
		Node: node,

		NTP: ntp,

		ChainStateTimeout:         180,
		ChainStateBroadcastPeriod: 30,

		TransactionPool: transactionPool,
		Stake:           stake,

		BaseDir:            path.Join(userCurrentDir.HomeDir, ".zond"),
		ChainFileDirectory: "data",

		//API: api,
		//MongoProcessorConfig: mongoProcessorConfig,
	}

	return userConf
}

func (u *UserConfig) DataDir() string {
	return path.Join(u.BaseDir, u.ChainFileDirectory)
}

func (u *UserConfig) SetDataDir(dataDir string) {
	u.BaseDir = dataDir
}

func (u *UserConfig) GetLogFileName() string {
	return path.Join(u.BaseDir, "zond-daemon.log")
}

func GetDevConfig() (dev *DevConfig) {
	genesis := &GenesisConfig{
		GenesisPrevHeaderHash: []byte("Outside Context Problem"),
		MaxCoinSupply:         105000000000000000,
		SuppliedCoins:         65000000000000000,
		GenesisDifficulty:     10000000,
		CoinBaseAddress:       misc.HStr2Bin("0000000000000000000000000000000000000000000000000000000000000000"),
		GenesisTimestamp:      1600257256,
	}
	transaction := &TransactionConfig{
		MultiOutputLimit: 100,
	}

	dev = &DevConfig{
		Genesis: genesis,

		Version: "0.0.1 go",

		BlocksPerEpoch:       100,
		BlockLeadTimestamp:   30,
		BlockMaxDrift:        15,
		MaxFutureBlockLength: 256,
		MaxMarginBlockNumber: 32,
		MinMarginBlockNumber: 7,

		ReorgLimit: 22000,

		MessageQSize:		   300,
		MessageReceiptTimeout: 10,
		MessageBufferSize:     64 * 1024 * 1024,

		OTSBitFieldPerPage: 8192 / 8,

		DefaultNonce:            0,
		DefaultAccountBalance:   0,
		BlockTime: 60,

		DBName:              "state",
		PeersFilename:       "peers.json",
		WalletDatFilename:   "wallet.json",
		BannedPeersFilename: "banned_peers",

		Transaction: transaction,

		NumberOfBlockAnalyze: 10,
		SizeMultiplier:       1.1,
		BlockMinSizeLimit:    1024 * 1024,
		TxExtraOverhead:      15,

		ShorPerQuanta: 1000000000,

		MaxReceivableBytes: 10 * 1024 * 1024,
		ReservedQuota: 1024,

		BlockTimeSeriesSize: 1440,
		RecordTransactionHashes: false,
	}
	dev.MaxBytesOut = dev.MaxReceivableBytes - dev.ReservedQuota
	return dev
}