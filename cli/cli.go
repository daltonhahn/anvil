package cli

import (
	"log"
	"net/http"
	"fmt"
	"os"
	"github.com/thatisuday/commando"
	"github.com/daltonhahn/anvil/anvil"
	//"github.com/daltonhahn/anvil/security"
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
		AddFlag("server,s", "Registers this node as part of the Raft consensus protocol", commando.Bool, nil).
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			servFlag, _ := flags["server"].GetBool()
			if servFlag == true {
				anvil.AnvilInit("server")
			} else {
				anvil.AnvilInit("client")
			}
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
				hname, err := os.Hostname()
				if err != nil {
					log.Fatalln("Unable to get hostname")
				}
				_, err = http.Get("http://" + hname + ":443/anvil/catalog/nodes")
				//_, err = security.TLSGetReq(hname, "/anvil/catalog/nodes")
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
				hname, err := os.Hostname()
				if err != nil {
					log.Fatalln("Unable to get hostname")
				}
				//_, err = security.TLSGetReq(hname, "/anvil/catalog/services")
				_, err = http.Get("http://" + hname + ":443/anvil/catalog/services")
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				log.Fatalln("Anvil binary is not currently running")
			}
		})

	commando.
		Register("peers").
		SetDescription("This command is utilized to retrieve a list of raft peers within Anvil.").
		SetShortDescription("lists raft peers of anvil service mesh node").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			//Check if Anvil binary is running
			res := anvil.CheckStatus()
			if (res == true) {
				hname, err := os.Hostname()
				if err != nil {
					log.Fatalln("Unable to get hostname")
				}
				//_, err = security.TLSGetReq(hname, "/anvil/raft/peers")
				_, err = http.Get("http://" + hname + ":443/anvil/raft/peers")
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				log.Fatalln("Anvil binary is not currently running")
			}
		})

	commando.
		Register("acl").
		SetDescription("This command is utilized to push an ACL object into the raft log of Anvil.").
		SetShortDescription("add an ACL object to the service mesh's raft log").
		AddArgument("acl command", "Anvil ACL command to execute", "").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
                        //Check if Anvil binary is running
                        res := anvil.CheckStatus()
                        if (res == true) {
                                for _,v := range args {
                                        anvil.Submit(v.Value)
                                }
                        } else {
                                log.Fatalln("Anvil binary is not currently running")
                        }
                })

	// parse command-line arguments from the STDIN
	commando.Parse(nil)
}
