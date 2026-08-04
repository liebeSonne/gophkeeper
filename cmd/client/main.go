package main

import (
	"github.com/liebeSonne/gophkeeper/internal/client/cmd"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	cmd.SetVersion(buildVersion, buildDate, buildCommit)
	cmd.Setup()
	cmd.Execute()
}
