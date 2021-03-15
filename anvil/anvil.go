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
	sb := string(body)
	fmt.Println("GOT THIS BACK FROM POST ENDPOINT: ", sb)

	//Take all of your content and POST it to the register endpoint of
	//the target node

	//Trigger HTTP request to target node /catalog/register endpoint
	//If target responds with 200 OK, add target to your own catalog
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
		fmt.Printf("Some error %v\n", err)
		return
	}

	go gossip.HandleUDP(p, ser)

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

	// Unmarshal
	var msg Message
	err = json.Unmarshal(b, &msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprint(w, "Hit the register endpoint\n")

	//Replace me
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

