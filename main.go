package main

import (
	"github.com/daltonhahn/anvil/cli"
	"github.com/daltonhahn/anvil/logging"
)

func main() {
	logging.InitLog()
	cli.CLI()
}
