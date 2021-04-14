package router

import (
	"io/ioutil"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	//"errors"

	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/security"
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
	hname, err := os.Hostname()
        if err != nil {
                log.Fatalln("Unable to get hostname")
        }
	resp, err := security.TLSGetReq(hname, "/anvil/catalog")
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

func RaftPeers(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Raft Peers at " + dt.String() + "\n"))
	raft.GetPeers()
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

	reply := raft.RequestVote(rv_args)
	var jsonData []byte

	jsonData, err = json.Marshal(reply)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(jsonData))
}

func AppendEntries(w http.ResponseWriter, r *http.Request) {
        b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
        var ae_args raft.AppendEntriesArgs
        err = json.Unmarshal(b, &ae_args)
        if err != nil {
                log.Fatalln(err)
                return
        }

        reply := raft.AppendEntries(ae_args)
        var jsonData []byte

        jsonData, err = json.Marshal(reply)
        if err != nil {
                log.Fatalln("Unable to marshal JSON")
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, string(jsonData))
}

func UpdateLeader(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	var newLead map[string]string
	err = json.Unmarshal(b, &newLead)
	catalog.UpdateNodeTypes(newLead["leader"])
	fmt.Fprintf(w, "OK")
}

func CatchOutbound(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Host, " ", r.Method, " ", r.RequestURI)
	/*resp,err := (*http.Response, errors.New("No Response"))
	if (r.Method == "POST") {
		security.TLSPostReq(r.Host, )
		TLSPostReq(target string, path string, options string, body io.Reader) (*http.Response, error) {
	}
	else {
		security.TLSGetReq()
		func TLSGetReq(target string, path string) (*http.Response,error) {
	}
	*/
}
