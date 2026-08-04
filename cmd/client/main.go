package main

import (
	"github.com/liebeSonne/gophkeeper/internal/client/app"
	"github.com/liebeSonne/gophkeeper/internal/client/cmd"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	defer func() {
		_ = app.Close()
	}()

	cmd.SetVersion(buildVersion, buildDate, buildCommit)
	cmd.Setup()
	cmd.Execute()
}
