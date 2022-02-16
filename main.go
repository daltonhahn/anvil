package main

import (
	"github.com/daltonhahn/anvil/cli"
	"github.com/daltonhahn/anvil/logging"
)

func main() {
	logging.InitLog()
	logging.InfoLogger.Println("Starting the Anvil binary. . .")
	cli.CLI()
}
