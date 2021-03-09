package main

import (
	"os"
	"os/exec"
	"log"
	"net/http"
	"github.com/daltonhahn/anvil/iptables"
	"github.com/daltonhahn/anvil/envoy"
	"github.com/daltonhahn/anvil/anvil"

	"github.com/julienschmidt/httprouter"
)

func main() {
	envoy.SetupEnvoy()
	cmd := &exec.Cmd {
		Path: "/usr/bin/envoy",
		Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy.yaml" },
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	cmd.Start()

	iptables.MakeIpTables()

	router := httprouter.New()
	router.GET("/", anvil.Index)
	router.GET("/catalog", anvil.GetCatalog)
	router.GET("/catalog/:node", anvil.GetNodeServices)

	log.Fatal(http.ListenAndServe(":8080", router))
	cmd.Wait()
}
