package cli

import (
	"fmt"
	"github.com/thatisuday/commando"
	"github.com/daltonhahn/anvil/anvil"
)

func CLI() {
	commando.
		SetExecutableName("anvil").
		SetVersion("v0.0.1").
		SetDescription("Anvil --  A research-oriented service mesh for security")

	commando.
		Register(nil).
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			fmt.Println("Welcome to Anvil Service Mesh\n")
		})

	commando.
		Register("start").
		SetDescription("This command initializes and begins the Anvil service mesh processes.").
		SetShortDescription("runs anvil service mesh").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			anvil.AnvilInit()
		})

	// parse command-line arguments from the STDIN
	commando.Parse(nil)
}
