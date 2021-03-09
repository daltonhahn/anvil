package anvil

import (
	"fmt"
	"time"
	"os"
	"os/exec"
	"log"
	"net/http"
	"github.com/julienschmidt/httprouter"

	"github.com/daltonhahn/anvil/envoy"
	"github.com/daltonhahn/anvil/iptables"
)

func AnvilInit() {
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
        router.GET("/", Index)
        router.GET("/catalog", GetCatalog)
        router.GET("/catalog/:node", GetNodeServices)

        log.Fatal(http.ListenAndServe(":8080", router))
        cmd.Wait()
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func GetCatalog(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Catalog at " + dt.String() + "\n"))
}

func GetNodeServices(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node_name := params.ByName("node")
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Services for Node: " + node_name + "\nCurrent Time: " + dt.String() + "\n"))
}
