package anvil

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"net/http"
	"io/ioutil"
	"sync"
	"context"
	"time"
	_ "net/http/pprof"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/daltonhahn/anvil/anvil/gossip"
	"github.com/daltonhahn/anvil/network"
	"github.com/daltonhahn/anvil/router"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/logging"
	"github.com/daltonhahn/anvil/service"
)

var RotationFlag bool
var NodeType string
var ConfigDir string
var DataDir string

var rotationTrigger bool

func Cleanup() {
    logging.InfoLogger.Println("Caught Ctrl+C, cleaning up and ending")
	network.RestoreIpTables(DataDir)
}

func readServiceConfig() (*struct{Services []service.Service}, error) {
	yamlFile, err := ioutil.ReadFile(ConfigDir + "/service.yaml")
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
        S_list, err := readServiceConfig()
        if err != nil {
                log.Fatal(err)
        }
        return S_list.Services
}

func AnvilInit(nodeType string, securityFlag bool, configDir string, dataDir string, profiling bool, rotationFlag bool) {
	DataDir = dataDir
	ConfigDir = configDir
	NodeType = nodeType
	RotationFlag = rotationFlag

	logging.InitLog(DataDir)
	if NodeType == "server" {
		logging.QuorumLogInit()
	}
	logging.CatalogLogInit()
	logging.InfoLogger.Println("Starting the Anvil binary. . .")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        Cleanup()
		wg.Done()
        os.Exit(1)
    }()
	logging.LogStartMessage(ConfigDir, DataDir, securityFlag, nodeType, profiling, RotationFlag)

	if network.CheckTables() {
		network.SaveIpTables(DataDir)
		if !network.MakeIpTables() {
			Cleanup()
			wg.Done()
			os.Exit(1)
		}
	}
	
	if securityFlag {
		security.ReadSecConfig()
	}

	hname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	network.SetHosts(hname)
	wg.Wait()

	serviceMap := SetServiceList()
	catalog.Register(hname, serviceMap, nodeType)

	if nodeType == "server" {
		_ = raft.NewConsensusModule(hname, []string{""})
	}

	wg = new(sync.WaitGroup)
	wg.Add(2)
	anv_router := mux.NewRouter()
	svc_router := mux.NewRouter()
	registerRoutes(anv_router)
	registerSvcRoutes(svc_router)

	// Separate Server starts into different function eventually

	registerUDP()


	cw, err := New()
	if err != nil {
			log.Println(err)
	}
	if err := cw.Watch(); err != nil {
			log.Println(err)
	}
	rotationTrigger = true
	go func() {
		for {
			<-sigHandle
			if rotationTrigger == true {
				rotationTrigger = false
				ctxShutDown, cancel := context.WithTimeout(context.Background(), (5*time.Second))
				defer func() {
					cancel()
				}()

				if err := server.Shutdown(ctxShutDown); err != nil {
					log.Printf("server Shutdown Failed:%+s", err)
				}
				go cw.startNewServer(anv_router)
			}
		}
		wg.Done()
	}()
	go cw.startNewServer(anv_router)

	go func() {
		log.Fatal(http.ListenAndServe(":444", svc_router))
		wg.Done()
	}()

	if profiling {
		wg.Add(1)
		go func() {
			log.Println(http.ListenAndServe(":6666", nil))
			wg.Done()
		}()
	}

	wg.Wait()
}

func registerUDP() {
	// Add some logging
	p := make([]byte, 4096)
	ser, err := net.ListenUDP("udp", &net.UDPAddr{Port: 443, IP: net.IP([]byte("0.0.0.0"))})
	if err != nil {
		log.Fatalf("Some error %v\n", err)
	}
	go gossip.HandleUDP(p, ser)
	go gossip.CheckHealth(ser)
	go gossip.PropagateCatalog(ser)
}

func registerRoutes(anv_router *mux.Router) {
	anv_router.HandleFunc("/anvil/raft/requestvote", router.RequestVote).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/appendentries", router.AppendEntries).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/peers", router.RaftPeers).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/peerList", router.RaftGetPeers).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/updateleader", router.UpdateLeader).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/backlog/{index}", router.RaftBacklog).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/pushACL", router.PushACL).Methods("POST")
	anv_router.HandleFunc("/anvil/raft/getACL", router.GetACL).Methods("GET")
	anv_router.HandleFunc("/anvil/raft/acl/{service}", router.TokenLookup).Methods("POST")
	anv_router.HandleFunc("/anvil/catalog/clients", router.GetCatalogClients).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog/nodes", router.GetNodeCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog/services", router.GetServiceCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog/register", router.RegisterNode).Methods("POST")
	anv_router.HandleFunc("/anvil/catalog/leader", router.GetCatalogLeader).Methods("GET")
	anv_router.HandleFunc("/anvil/catalog", router.GetCatalog).Methods("GET")
	anv_router.HandleFunc("/anvil/rotation/config", router.HandleConfigChange).Methods("GET")
	anv_router.HandleFunc("/anvil/rotation", router.HandleRotation).Methods("POST")
	anv_router.HandleFunc("/anvil/type", router.GetType).Methods("GET")
	anv_router.HandleFunc("/anvil/", router.Index).Methods("GET")
	anv_router.PathPrefix("/service/{query}").HandlerFunc(router.RerouteService).Methods("GET","POST")
}

func registerSvcRoutes(svc_router *mux.Router) {
	svc_router.PathPrefix("/outbound/{query}").HandlerFunc(router.CatchOutbound).Methods("GET","POST")
}
