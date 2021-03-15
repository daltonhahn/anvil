package cli

import (
	"log"
	"net/http"
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

	commando.
		Register("join").
		SetDescription("This command is utilized to trigger a registration of nodes within Anvil.").
		SetShortDescription("joins anvil service mesh nodes").
		AddArgument("target node", "Anvil node to join", "").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			//Check if Anvil binary is running
			res := anvil.CheckStatus()
			if (res == true) {
				for _,v := range args {
					anvil.Join(v.Value)
				}
			} else {
				log.Fatalln("Anvil binary is not currently running")
			}
		})

	commando.
		Register("nodes").
		SetDescription("This command is utilized to retrieve a list of available nodes within Anvil.").
		SetShortDescription("lists anvil service mesh nodes").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			//Check if Anvil binary is running
			res := anvil.CheckStatus()
			if (res == true) {
				_, err := http.Get("http://localhost/anvil/catalog/nodes")
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				log.Fatalln("Anvil binary is not currently running")
			}
		})

	commando.
		Register("services").
		SetDescription("This command is utilized to retrieve a list of available services within Anvil.").
		SetShortDescription("lists anvil service mesh services").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			//Check if Anvil binary is running
			res := anvil.CheckStatus()
			if (res == true) {
				_, err := http.Get("http://localhost/anvil/catalog/services")
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				log.Fatalln("Anvil binary is not currently running")
			}
		})

	// parse command-line arguments from the STDIN
	commando.Parse(nil)
}
