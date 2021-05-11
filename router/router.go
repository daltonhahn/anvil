package router

import (
	"io/ioutil"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	//"strings"
	//"errors"

	"github.com/gorilla/mux"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	//"github.com/daltonhahn/anvil/security"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []catalog.Service `json:"services"`
	NodeType string `json:"nodetype"`
}

type G_Message struct {
        NodeName string `json:"nodename"`
        Iteration int64 `json:"iteration"`
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
	var msg G_Message
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
	resp, err := http.Get("http://" + hname + ":443/anvil/catalog")
	//resp, err := security.TLSGetReq(hname, "/anvil/catalog")
        if err != nil {
                log.Fatalln("Unable to get response")
        }

        body, err := ioutil.ReadAll(resp.Body)
        var receivedStuff G_Message

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
	iteration := anv_catalog.GetIter()
	nodes := []catalog.Node(anv_catalog.GetNodes())
	services := []catalog.Service(anv_catalog.GetServices())
	nodeType := anv_catalog.GetNodeType(hname)
	newMsg := &G_Message{hname, iteration, nodes, services, nodeType}
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

func PushACL(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	var comm map[string]string
	err = json.Unmarshal(b, &comm)
	raft.Submit(comm["command"])
}

func GetACL(w http.ResponseWriter, r *http.Request) {
        dt := time.Now()
        fmt.Fprint(w, ("Retrieving ACL Log at " + dt.String() + "\n"))
        raft.GetLog()
}

func TokenLookup(w http.ResponseWriter, r *http.Request) {
        dt := time.Now()
	b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	var token_meta map[string]string
	err = json.Unmarshal(b, &token_meta)
	fmt.Fprint(w, ("Retrieving ACL Token data for: " + token_meta["id"] + " at " + dt.String() + "\n"))
	//result := raft.TokenLookup(token_meta["id"], token_meta["svc"], token_meta["requester"], token_meta["time"])
	result := raft.TokenLookup("test")
	fmt.Fprintf(w, strconv.FormatBool(result))
}

func GetIterCatalog(w http.ResponseWriter, r *http.Request) {
	target := mux.Vars(r)["node"]
	anv_catalog := catalog.GetCatalog()
	val := anv_catalog.GetNodeIter(target)
	fmt.Fprintf(w, strconv.FormatInt(val,10))
}

func UpdateIter(w http.ResponseWriter, r *http.Request) {
	target := mux.Vars(r)["node"]
        b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
        var newIter G_Message
        err = json.Unmarshal(b, &newIter)
	anv_catalog := catalog.GetCatalog()
        anv_catalog.UpdateIter(target, newIter.Iteration)
        fmt.Fprintf(w, "OK")
}

func RaftBacklog(w http.ResponseWriter, r *http.Request) {
	strIndex := mux.Vars(r)["index"]
	fmt.Println("Got backlog request for index: ", strIndex)
	index, _ := strconv.ParseInt(strIndex, 10, 64)
	entries := raft.PullBacklogEntries(index)
	//Add some logic to put list of log entries in JSON object to give back
	fmt.Println("Missing log entries: ", entries)
}

/*
func CatchOutbound(w http.ResponseWriter, r *http.Request) {
	var resp *http.Response
	var err error
	fmt.Println("CATCHING OUTBOUND")
	fmt.Println("LOOKING FOR: ", r.Host, " AT PATH: ", r.RequestURI)
	fmt.Println("Original DST: ", r.Header.Get("X-Envoy-Original-Dst-Host"))

	if (r.Method == "POST") {
		resp, err = security.TLSPostReq(r.Host, r.RequestURI[strings.Index(r.RequestURI,"/outbound")+9:], r.Header.Get("Content-Type"), r.Body)
	} else {
		resp, err = security.TLSGetReq(r.Host, r.RequestURI[strings.Index(r.RequestURI,"/outbound")+9:])
	}
	if err != nil {
		fmt.Fprintf(w, "Bad Response")
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Bad Read")
	}

	fmt.Println(string(respBody))
	//fmt.Fprintf(w, string(respBody))
	fmt.Fprintf(w, "Rerouting -------\n")
}
*/
