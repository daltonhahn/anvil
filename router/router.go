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
	//"errors"

	"github.com/gorilla/mux"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/service"
	"github.com/daltonhahn/anvil/acl"
	"github.com/daltonhahn/anvil/security"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []service.Service `json:"services"`
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
	//resp, err := http.Get("http://" + hname + ":443/anvil/catalog")
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
	services := []service.Service(anv_catalog.GetServices())
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
func GetCatalogLeader(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	leader := anv_catalog.GetLeader()
	fmt.Fprint(w, (leader))
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
	var aclObj acl.ACLEntry
	err = json.Unmarshal(b, &aclObj)
	raft.Submit(aclObj)
}

func GetACL(w http.ResponseWriter, r *http.Request) {
        dt := time.Now()
        fmt.Fprint(w, ("Retrieving ACL Log at " + dt.String() + "\n"))
        raft.GetLog()
}

func TokenLookup(w http.ResponseWriter, r *http.Request) {
	serviceTarget := mux.Vars(r)["service"]
        dt := time.Now()
	b, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	var lookupDat string
	err = json.Unmarshal(b, &lookupDat)
	fmt.Fprint(w, ("Retrieving ACL Token status at " + dt.String() + "\n"))
	result := raft.TokenLookup(lookupDat, serviceTarget, time.Now())
	fmt.Fprintf(w, strconv.FormatBool(result))
}

func RaftBacklog(w http.ResponseWriter, r *http.Request) {
	strIndex := mux.Vars(r)["index"]
	index, _ := strconv.ParseInt(strIndex, 10, 64)
	entries := raft.PullBacklogEntries(index)
	var jsonData []byte
	jsonData, err := json.Marshal(entries)
        if err != nil {
                log.Fatalln("Unable to marshal JSON")
        }
	w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, string(jsonData))
}

func CatchOutbound(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI[9:])
	var resp *http.Response
	var err error
	anv_catalog := catalog.GetCatalog()
	target := anv_catalog.GetSvcHost(r.Host)
	if (r.Method == "POST") {
		resp, err = security.TLSPostReq(target, r.RequestURI[9:], r.Header.Get("Content-Type"), r.Body)
	} else {
		resp, err = security.TLSGetReq(target, r.RequestURI[9:])
	}
	if err != nil {
		fmt.Fprintf(w, "Bad Response")
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Bad Read")
	}

	fmt.Fprintf(w, string(respBody))
}

func RerouteService(w http.ResponseWriter, r *http.Request) {
	var resp *http.Response
	var err error
	anv_catalog := catalog.GetCatalog()
	target_port := anv_catalog.GetSvcPort(r.RequestURI[9:])
	if (r.Method == "POST") {
		// Add more URI Processing in-case Index page isn't the one called
		resp, err = http.Post("http://"+r.Host+":"+strconv.FormatInt(target_port,10), r.Header.Get("Content-Type"), r.Body)
	} else {
		// Add more URI Processing in-case Index page isn't the one called
		resp, err = http.Get("http://"+r.Host+":"+strconv.FormatInt(target_port,10))
	}
	if err != nil {
		fmt.Fprintf(w, "Bad Response")
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Bad Read")
	}

	fmt.Fprintf(w, string(respBody))
}

