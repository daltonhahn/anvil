package anvil

import (
	"os"
	"log"
	"net"
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"sync"

	"github.com/daltonhahn/anvil/network"
	"github.com/daltonhahn/anvil/router"
	"github.com/daltonhahn/anvil/anvil/gossip"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/service"
)

func readEnvoyConfig() (*struct{Services []service.Service}, error) {
        yamlFile, err := ioutil.ReadFile("/root/anvil/config/services/sample-svc.yaml")
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
	var S_list struct {
		Services	[]service.Service
	}
        err = yaml.Unmarshal(yamlFile, &S_list)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }

        return &S_list, nil
}

func SetServiceList() ([]service.Service) {
        S_list, err := readEnvoyConfig()
        if err != nil {
                log.Fatal(err)
        }
        return S_list.Services
}

func AnvilInit(nodeType string) {
	network.CleanTables()
        network.MakeIpTables()
	security.ReadSecConfig()

	hname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	network.SetHosts(hname)
	serviceMap := SetServiceList()
	catalog.Register(hname, serviceMap, nodeType)

	if nodeType == "server" {
		CM := raft.NewConsensusModule(hname, []string{""})
		fmt.Println(CM)
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)
        anv_router := mux.NewRouter()
        svc_router := mux.NewRouter()
	registerRoutes(anv_router)
	registerSvcRoutes(svc_router)
	registerUDP()
	go func() {
		log.Fatal(http.ListenAndServeTLS(":443", "/root/anvil/config/certs/test1.crt", "/root/anvil/config/certs/test1.key", anv_router))
		wg.Done()
	}()
	go func() {
		log.Fatal(http.ListenAndServe(":444", svc_router))
		wg.Done()
	}()
	wg.Wait()
}

func registerUDP() {
	p := make([]byte, 4096)
	addr := net.UDPAddr{
		Port: 443,
		IP: net.ParseIP("0.0.0.0"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalln("Some error %v\n", err)
	}
	go gossip.HandleUDP(p, ser)
	go gossip.CheckHealth()
	go gossip.PropagateCatalog()
}

func registerRoutes(anv_router *mux.Router) {
	anv_router.HandleFunc("/anvil/raft/requestvote", router.RequestVote).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/appendentries", router.AppendEntries).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/peers", router.RaftPeers).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/updateleader", router.UpdateLeader).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/backlog/{index}", router.RaftBacklog).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/pushACL", router.PushACL).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/getACL", router.GetACL).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/acl/{service}", router.TokenLookup).Methods("POST")
	anv_router.HandleFunc("/anvil/catalog/nodes", router.GetNodeCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog/services", router.GetServiceCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog/register", router.RegisterNode).Methods("POST")
	anv_router.HandleFunc("/anvil/catalog/leader", router.GetCatalogLeader).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog", router.GetCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/", router.Index).Methods("GET")
	anv_router.HandleFunc("/service/{query}", router.RerouteService).Methods("GET","POST")
}

func registerSvcRoutes(svc_router *mux.Router) {
	svc_router.PathPrefix("/outbound/{query}").HandlerFunc(router.CatchOutbound).Methods("GET","POST")
}
