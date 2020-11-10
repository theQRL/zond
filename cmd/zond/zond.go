package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/consensus"
	"github.com/theQRL/zond/db"
	"github.com/theQRL/zond/p2p"
	"github.com/theQRL/zond/state"
	"io"
	"os"
	"os/signal"
)

var (
	publicAPIServer *api.PublicAPIServer
)

func ConfigCheck() bool {
	return true
}

func run(c *chain.Chain, db *db.DB) {
	srv := p2p.NewServer(c)
	go srv.Start()
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
	logFile, err := os.OpenFile(config.GetUserConfig().GetLogFileName(),
		os.O_WRONLY | os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	return nil
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
	run(c, s.DB())

	/*
	1. Peer Tries to Connect 10 peer ips 10 with a delay of 10 seconds until max peer limit is reached
	2. OnConnection the ip address is checked in the map, if exists, then close the connection
	3. 
	 */
	log.Info("Shutting Down Node")
}