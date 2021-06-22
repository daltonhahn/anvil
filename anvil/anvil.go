package anvil

import (
	"os"
	"log"
	"net"
	"net/http"
	"github.com/gorilla/mux"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"sync"
	"crypto/tls"
	//"crypto/x509"
	"fmt"

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
		fmt.Println("Unable to read sample-svc.yaml")
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

	//
	cw, err := New()
        if err := cw.Watch(); err != nil {
                fmt.Println(err)
        }
	//


	/*
	caCert, err := ioutil.ReadFile(security.SecConf[0].CACert)
        if err != nil {
		fmt.Println("Unable to read config 1 ca.crt")
                log.Printf("Read file error #%v", err)
        }
        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		ClientAuth:		tls.RequireAndVerifyClientCert,
		ClientCAs:		caCertPool,
	}

	if len(security.SecConf) >= 2 {
		tlsConfig.Certificates = make([]tls.Certificate, 2)
	} else {
		tlsConfig.Certificates = make([]tls.Certificate, 1)
	}

	tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(security.SecConf[0].TLSCert, security.SecConf[0].TLSKey)
	if err != nil {
		log.Fatal(err)
	}
	if len(security.SecConf) >= 2 {
		caCert, err := ioutil.ReadFile(security.SecConf[1].CACert)
		if err != nil {
			fmt.Println("Unable to read config 2 ca.crt")
			log.Printf("Read file error #%v", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.Certificates[1], err = tls.LoadX509KeyPair(security.SecConf[1].TLSCert, security.SecConf[1].TLSKey)
		if err != nil {
			log.Fatal(err)
		}
	}
	tlsConfig.BuildNameToCertificate()
	*/

	hname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	network.SetHosts(hname)
	serviceMap := SetServiceList()
	catalog.Register(hname, serviceMap, nodeType)

	if nodeType == "server" {
		_ = raft.NewConsensusModule(hname, []string{""})
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)
        anv_router := mux.NewRouter()
        svc_router := mux.NewRouter()
	registerRoutes(anv_router)
	registerSvcRoutes(svc_router)
	registerUDP()

	dump, err := os.OpenFile("/dev/null", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open dev null")
		log.Println(err)
	}
	nullLog := log.New(dump, "", log.LstdFlags)
	server := &http.Server{
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,
		Handler:	anv_router,
		ErrorLog:	nullLog,
	}

	    listener, err := tls.Listen("tcp", ":443", tlsConfig)
	    if err != nil {
		    log.Fatal(err)
	    }

	go func() {
		server.Serve(listener)
		/*
		fmt.Println("Starting first config server")
		err1 := http.ListenAndServeTLS(":443", security.SecConf[0].TLSCert, security.SecConf[0].TLSKey, anv_router)
		if err1 != nil {
			fmt.Println("Got an error on first config")
			if len(security.SecConf) < 2 {
				fmt.Println("Killing on first error")
				log.Fatal(err1)
			} else {
				fmt.Println("Trying with second config")
				err2 := http.ListenAndServeTLS(":443", security.SecConf[1].TLSCert, security.SecConf[1].TLSKey, anv_router)
				if err2 != nil {
					fmt.Println("Killing on second error")
					log.Fatal(err2)
				}
			}
		}
		*/
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
	anv_router.HandleFunc("/anvil/", router.Index).Methods("GET")
	anv_router.PathPrefix("/service/{query}").HandlerFunc(router.RerouteService).Methods("GET","POST")
}

func registerSvcRoutes(svc_router *mux.Router) {
	svc_router.PathPrefix("/outbound/{query}").HandlerFunc(router.CatchOutbound).Methods("GET","POST")
}
