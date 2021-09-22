package router

import (
	"io/ioutil"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"bytes"
	"errors"

	"github.com/gorilla/mux"
	"github.com/avast/retry-go/v3"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/service"
	"github.com/daltonhahn/anvil/acl"
	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/rotation"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []service.Service `json:"services"`
	NodeType string `json:"nodetype"`
}

func RegisterNode(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
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
	resp, err := security.TLSGetReq(hname, "/anvil/catalog", "")
        if err != nil {
                log.Println("Unable to get response")
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

func GetCatalogClients(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	clients := anv_catalog.GetClients()
	retList := struct {
		Clients		[]string
	}{Clients: clients}
	var jsonData []byte
	jsonData, err := json.Marshal(retList)
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

func RaftGetPeers(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	peerList := anv_catalog.GetPeers()

	jsonData, err := json.Marshal(peerList)
        if err != nil {
                log.Fatalln("Unable to marshal JSON")
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, string(jsonData))
}

func RaftPeers(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Raft Peers at " + dt.String() + "\n"))
	raft.GetPeers()
}

func GetType(w http.ResponseWriter, r *http.Request) {
	anv_catalog := catalog.GetCatalog()
	hname, _ := os.Hostname()
	nodeType := anv_catalog.GetNodeType(hname)
	if nodeType == "server" || nodeType == "leader" {
		fmt.Fprint(w, "server")
	} else {
		fmt.Fprint(w, "client")
	}
}

func RequestVote(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
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
        r.Body.Close()
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
        r.Body.Close()
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
        r.Body.Close()
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
	b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	var lookupDat string
	err = json.Unmarshal(b, &lookupDat)
	result := raft.TokenLookup(lookupDat, serviceTarget, time.Now())
	fmt.Fprintf(w, strconv.FormatBool(result))
}

func HandleRotation(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err != nil {
                http.Error(w, err.Error(), 500)
                return
        }
	collectTags := struct {
		Iteration	string
		Prefix		string
		QuorumMems	[]string
	}{}
	err = json.Unmarshal(b, &collectTags)
	if err != nil {
		log.Println("Unable to unmarshal the data that was sent in this post request")
	}
	result := rotation.CollectFiles(collectTags.Iteration, collectTags.Prefix, collectTags.QuorumMems)
	if result == true {
		fmt.Fprintf(w, strconv.FormatBool(result))
	} else {
		http.NotFound(w, r)
	}
}

func HandleConfigChange(w http.ResponseWriter, r *http.Request) {
	rotation.AdjustConfig()
	fmt.Fprintf(w, "OK")
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
	fmt.Println("Landed in CatchOutbound")
	var resp *http.Response
	var err error
	var body []byte
	anv_catalog := catalog.GetCatalog()
	target := anv_catalog.GetSvcHost(r.Host)
	target_uri := "/"+strings.Join(strings.Split(r.RequestURI, "/")[3:], "/")
	if (r.Method == "POST") {
                err = retry.Do(
                        func() error {
				fmt.Println("Trying to make POST request to %s", target)

				resp, err = security.TLSPostReq(target, target_uri, strings.Split(r.RequestURI, "/")[2], r.Header.Get("Content-Type"), r.Body)
                                if err != nil || resp.StatusCode != http.StatusOK {
                                        if err == nil {
                                                return errors.New("BAD STATUS CODE FROM SERVER")
                                        } else {
                                                return err
                                        }
                                } else {
                                        body, err = ioutil.ReadAll(resp.Body)
                                        resp.Body.Close()
                                        if err != nil {
                                                return err
                                        }
                                        return nil
                                }
                        },
                        retry.Attempts(3),
                )
	} else {
		err = retry.Do(
                        func() error {
				fmt.Println("Trying to make GET request to ", target, target_uri, strings.Split(r.RequestURI, "/")[2])

				resp, err = security.TLSGetReq(target, target_uri, strings.Split(r.RequestURI, "/")[2])
                                if err != nil || resp.StatusCode != http.StatusOK {
                                        if err == nil {
                                                return errors.New("BAD STATUS CODE FROM SERVER")
                                        } else {
                                                return err
                                        }
                                } else {
                                        body, err = ioutil.ReadAll(resp.Body)
                                        resp.Body.Close()
                                        if err != nil {
                                                return err
                                        }
                                        return nil
                                }
                        },
                        retry.Attempts(3),
                )
		fmt.Println(err)
	}
	fmt.Fprintf(w, string(body))
}

func RerouteService(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Landed in reroute service")
        target_svc := strings.Split(r.RequestURI, "/")[2]
        tok_recv := r.Header["Authorization"][0]
        anv_catalog := catalog.GetCatalog()
        verifier := anv_catalog.GetQuorumMem()
        var resp *http.Response
        var err error
	var appbody []byte
        postBody, _ := json.Marshal(tok_recv)
        responseBody := bytes.NewBuffer(postBody)
	err = retry.Do(
		func() error {
			resp, err = security.TLSPostReq(verifier, "/anvil/raft/acl/"+target_svc, "", "application/json", responseBody)
			if err != nil || resp.StatusCode != http.StatusOK {
				if err == nil {
					return errors.New("BAD STATUS CODE FROM SERVER")
				} else {
					return err
				}
			} else {
				appbody, err = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return err
				}
				return nil
			}
		},
		retry.Attempts(3),
	)
	fmt.Println("Reroute error: ", err)
        approval, _ := strconv.ParseBool(string(appbody))
	fmt.Println(approval)
        if (approval) {
		var body []byte
                target_port := anv_catalog.GetSvcPort(strings.Split(r.RequestURI, "/")[2])
                rem_path := "/"+strings.Join(strings.Split(r.RequestURI, "/")[3:], "/")
		reqURL, _ := url.Parse("http://"+r.Host+":"+strconv.FormatInt(target_port,10)+rem_path)
		req := &http.Request {
			URL: reqURL,
			Method: r.Method,
			Header: r.Header,
			Body: r.Body,
		}
		req.Header.Add("X-Forwarded-For", strings.Split(r.RemoteAddr,":")[0])
                if (r.Method == "POST") {
			err = retry.Do(
				func() error {
					resp, err := http.DefaultClient.Do(req)
					if err != nil || resp.StatusCode != http.StatusOK {
						if err == nil {
							return errors.New("BAD STATUS CODE FROM SERVER")
						} else {
							return err
						}
					} else {
						body, err = ioutil.ReadAll(resp.Body)
						resp.Body.Close()
						if err != nil {
							return err
						}
						return nil
					}
				},
				retry.Attempts(3),
			)
			fmt.Fprintf(w, string(body))
		} else {
			//resp, err = http.Get("http://"+r.Host+":"+strconv.FormatInt(target_port,10)+rem_path)
			err = retry.Do(
				func() error {
					resp, err := http.DefaultClient.Do(req)
					if err != nil || resp.StatusCode != http.StatusOK {
						if err == nil {
							return errors.New("BAD STATUS CODE FROM SERVER")
						} else {
							return err
						}
					} else {
						body, err = ioutil.ReadAll(resp.Body)
						resp.Body.Close()
						if err != nil {
							return err
						}
						return nil
					}
				},
				retry.Attempts(3),
			)
			fmt.Fprintf(w, string(body))
		}
	} else {
		http.Error(w, "Token not validated", http.StatusForbidden)
	}
}
