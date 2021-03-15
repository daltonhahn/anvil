package anvil

import (
	"fmt"
	"time"
	"os"
	"os/exec"
	"log"
	"net"
	"net/http"
	"io/ioutil"
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"

	"github.com/daltonhahn/anvil/envoy"
	"github.com/daltonhahn/anvil/iptables"
	"github.com/daltonhahn/anvil/anvil/gossip"
	"github.com/daltonhahn/anvil/catalog"
)

type Message struct {
	NodeName string `json:"nodename"`
	Nodes []catalog.Node `json:"nodes"`
	Services []catalog.Service `json:"services"`
}

func CheckStatus() bool {
	resp, err := http.Get("http://localhost/anvil")
	if resp != nil && err == nil {
		return true
	} else {
		return false
	}
}


func Join(target string) {
	//Collect all of the current info you have from your local catalog
	resp, err := http.Get("http://localhost/anvil/catalog")
	if err != nil {
		log.Fatalln("Unable to get response")
	}

	body, err := ioutil.ReadAll(resp.Body)
	var receivedStuff Message

	err = json.Unmarshal(body, &receivedStuff)
	if err != nil {
		log.Fatalln("Unable to decode JSON")
	}

	postBody, _ := json.Marshal(receivedStuff)

	responseBody := bytes.NewBuffer(postBody)
	resp, err = http.Post("http://" + target + "/anvil/catalog/register", "application/json", responseBody)
	if err != nil {
		log.Fatalln("Unable to post content")
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Unable to read received content")
	}

	var respMsg Message
	err = json.Unmarshal(body, &respMsg)
	if err != nil {
		log.Fatalln("Unable to process response JSON")
	}

	var tempCatalog catalog.Catalog
	for _, ele := range respMsg.Nodes {
		tempCatalog.AddNode(ele)
		for _, svc := range respMsg.Services {
			if (ele.Address == svc.Address) {
				tempCatalog.AddService(svc)
			}
		}
		var localPost Message
		localPost.NodeName = ele.Name
		localPost.Services = tempCatalog.Services
		postBody, _ = json.Marshal(localPost)
		responseBody = bytes.NewBuffer(postBody)
		// Marshal the struct into a postable message
		http.Post("http://localhost/anvil/catalog/register", "application/json", responseBody)
		tempCatalog = catalog.Catalog{}
	}
}

func AnvilInit() {
        envoy.SetupEnvoy()
        cmd := &exec.Cmd {
                Path: "/usr/bin/envoy",
                Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy.yaml" },
                Stdout: os.Stdout,
                Stderr: os.Stdout,
        }
        cmd.Start()

	iptables.CleanTables()
        iptables.MakeIpTables()

	hname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	serviceMap := envoy.S_list
	catalog.Register(hname, serviceMap.Services)

        router := mux.NewRouter()
	router.HandleFunc("/catalog/nodes", GetNodeCatalog).Methods("GET")
	router.HandleFunc("/catalog/services", GetServiceCatalog).Methods("GET")
	router.HandleFunc("/catalog/register", RegisterNode).Methods("POST")
	router.HandleFunc("/catalog", GetCatalog).Methods("GET")
	router.HandleFunc("/", Index).Methods("GET")

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

        log.Fatal(http.ListenAndServe(":8080", router))

        cmd.Wait()
}

func RegisterNode(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var msg Message
	err = json.Unmarshal(b, &msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Pull your local info from catalog
	resp, err := http.Get("http://localhost/anvil/catalog")
        if err != nil {
                log.Fatalln("Unable to get response")
        }

        body, err := ioutil.ReadAll(resp.Body)
        var receivedStuff Message

        err = json.Unmarshal(body, &receivedStuff)
        if err != nil {
                log.Fatalln("Unable to decode JSON")
        }
	var jsonData []byte

	//Pass your catalog contents back to joiner
        jsonData, err = json.Marshal(receivedStuff)
        if err != nil {
                log.Fatalln("Unable to marshal JSON")
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, string(jsonData))

	// Add newly joined node to your local registry
	catalog.Register(msg.NodeName, msg.Services)
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Anvil Service Mesh Index\n")
}
func GetCatalog(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	hname, _ := os.Hostname()
	nodes := []catalog.Node(anv_catalog.GetNodes())
	services := []catalog.Service(anv_catalog.GetServices())
	newMsg := &Message{hname, nodes, services}
	var jsonData []byte
	jsonData, err := json.Marshal(newMsg)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(jsonData))
}
func GetNodeCatalog(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Anvil Nodes at " + dt.String() + "\n"))
	anv_catalog := catalog.GetCatalog()
	anv_catalog.PrintNodes()
}
func GetServiceCatalog(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Anvil Services at " + dt.String() + "\n"))
	anv_catalog := catalog.GetCatalog()
	anv_catalog.PrintServices()
}

