package main

import (
	"fmt"
	"log"
	"os"

	internallogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	logger, err := internallogger.New(internallogger.Config{Level: internallogger.InfoLevel, Writer: os.Stderr})
	if err != nil {
		log.Fatalf("error initializing logger: %v", err)
	}

	logger.Info("GophKeeper-server starting")

	os.Exit(0)
}
