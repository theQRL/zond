package main

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	crypto2 "github.com/theQRL/go-libp2p-qrl/crypto"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/consensus"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/p2p"
	"github.com/theQRL/zond/state"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"io/ioutil"
	"os"
	"os/signal"
	"time"
)

var (
	publicAPIServer *api.PublicAPIServer
)

func ConfigCheck() bool {
	return true
}

func run(c *chain.Chain, db *db.DB, keys crypto.PrivKey) error {
	srv, err := p2p.NewServer(c)
	if err != nil {
		log.Error("Failed to initialize server")
		return err
	}
	go srv.Start(keys)
	defer srv.Stop()

	if config.GetUserConfig().API.PublicAPI.Enabled {
		publicAPIServer = api.NewPublicAPIServer(c, srv.GetRegisterAndBroadcastChan())
		go publicAPIServer.Start()
	}

	pos := consensus.NewPOS(srv, c, db)

	go pos.Run()
	defer pos.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	return nil
}

func CreateDirectoryIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func SetLogOutput() error {
	rotateFileHook, err := misc.NewRotateFileHook(misc.RotateFileConfig{
		Filename:   config.GetUserConfig().GetLogFileName(),
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     28,
		Formatter: &log.JSONFormatter{
			TimestampFormat: time.RFC3339,
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize file rotate hook: %v", err)
		return err
	}

	log.SetOutput(colorable.NewColorableStdout())
	formatter := new(prefixed.TextFormatter)
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formatter.FullTimestamp = true
	formatter.DisableColors = false
	log.SetFormatter(formatter)

	log.AddHook(rotateFileHook)
	return nil
}

func loadP2PDilithiumKey(filePath string) (crypto.PrivKey, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		keys, _, err := crypto2.GenerateDilithiumKey(nil)
		if err != nil {
			return nil, err
		}
		data, err := keys.Raw()
		if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
			return nil, err
		}
		return keys, nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return crypto2.UnmarshalDilithiumPrivateKey(data)
}

func main() {
	userConfig := config.GetUserConfig()
	devConfig := config.GetDevConfig()

	err := CreateDirectoryIfNotExists(userConfig.DataDir())
	if err != nil {
		log.Error("Error creating data directory ", err.Error())
		return
	}

	err = SetLogOutput()
	if err != nil {
		log.Error("Error in Set Log Output ", err.Error())
		return
	}

	if !ConfigCheck() {
		log.Error("Invalid Config")
		return
	}

	crypto2.LoadAllExtendedKeyTypes()
	keys, err := loadP2PDilithiumKey(userConfig.GetAbsoluteNodeKeyFilePath())
	if err != nil {
		log.Error("Failed to loadP2PDilithiumKey ", err.Error())
		return
	}

	s, err := state.NewState(userConfig.DataDir(), devConfig.DBName)
	if err != nil {
		log.Error("Error while loading state ", err.Error())
		return
	}

	c := chain.NewChain(s)
	if err := c.Load(); err != nil {
		log.Error("Error loading chain state ", err.Error())
		return
	}
	log.Info("Main Chain Loaded Successfully")
	err = run(c, s.DB(), keys)
	if err != nil {
		log.Error("Initialization Error ", err.Error())
	}

	/*
		1. Peer Tries to Connect 10 peer ips 10 with a delay of 10 seconds until max peer limit is reached
		2. OnConnection the ip address is checked in the map, if exists, then close the connection
		3.
	*/
	log.Info("Shutting Down Node")
}
