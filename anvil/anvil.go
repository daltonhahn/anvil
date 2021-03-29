package anvil

import (
	"os"
	"os/exec"
	"log"
	"net"
	"net/http"
	"fmt"
	"github.com/gorilla/mux"

	"github.com/daltonhahn/anvil/envoy"
	"github.com/daltonhahn/anvil/router"
	"github.com/daltonhahn/anvil/anvil/gossip"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
)

func AnvilInit(nodeType string) {
        envoy.SetupEnvoy()
        cmd := &exec.Cmd {
                Path: "/usr/bin/envoy",
                Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy.yaml" },
                Stdout: os.Stdout,
                Stderr: os.Stdout,
        }
        cmd.Start()

	hname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	serviceMap := envoy.S_list
	catalog.Register(hname, serviceMap.Services, nodeType)

	CM := raft.NewConsensusModule(hname, []string{""})
	fmt.Println(CM)

        anv_router := mux.NewRouter()
	registerRoutes(anv_router)
	registerUDP()
        log.Fatal(http.ListenAndServe(":8080", anv_router))
        cmd.Wait()
}

func registerUDP() {
	p := make([]byte, 2048)
	addr := net.UDPAddr{
		Port: 8080,
		IP: net.ParseIP("0.0.0.0"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalln("Some error %v\n", err)
	}
	go gossip.HandleUDP(p, ser)
	go gossip.CheckHealth(ser)
}

func registerRoutes(anv_router *mux.Router) {
	anv_router.HandleFunc("/raft/requestvote", router.RequestVote).Methods("POST")
	anv_router.HandleFunc("/raft/appendentries", router.AppendEntries).Methods("POST")
	anv_router.HandleFunc("/catalog/nodes", router.GetNodeCatalog).Methods("GET")
	anv_router.HandleFunc("/catalog/services", router.GetServiceCatalog).Methods("GET")
	anv_router.HandleFunc("/catalog/register", router.RegisterNode).Methods("POST")
	anv_router.HandleFunc("/catalog", router.GetCatalog).Methods("GET")
	anv_router.HandleFunc("/", router.Index).Methods("GET")
}
