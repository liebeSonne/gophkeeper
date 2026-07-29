package main

import (
	"fmt"
	"log"

	"github.com/liebeSonne/gophkeeper/internal/config"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	logger, err := initLogger(cfg)
	if err != nil {
		log.Fatalf("error initializing logger: %v", err)
	}
	defer func() {
		err = logger.Sync()
		if err != nil {
			log.Fatalf("error syncing logger: %v", err)
		}
	}()

	logger.Info("GophKeeper-server starting",
		"server_address", cfg.ServerAddress,
		"https", cfg.EnableHTTPS,
	)
}
