package router

import (
	"io/ioutil"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []catalog.Service `json:"services"`
	NodeType string `json:"nodetype"`
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
	catalog.Register(msg.NodeName, msg.Services, msg.NodeType)
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Anvil Service Mesh Index\n")
}
func GetCatalog(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	hname, _ := os.Hostname()
	nodes := []catalog.Node(anv_catalog.GetNodes())
	services := []catalog.Service(anv_catalog.GetServices())
	nodeType := anv_catalog.GetNodeType(hname)
	newMsg := &Message{hname, nodes, services, nodeType}
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

func RequestVote(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
        var rv_args raft.RequestVoteArgs
        err = json.Unmarshal(b, &rv_args)
        if err != nil {
                log.Fatalln(err)
                return
        }

	fmt.Printf("\tREQUEST VOTE TERM: %+v\n", rv_args.Term)

	reply := raft.CM.RequestVote(rv_args)
	var jsonData []byte

	jsonData, err = json.Marshal(reply)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(jsonData))
}

func AppendEntries(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hit the appendentries endpoint")
}
