package gossip

import (
	"time"
	"net"
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"os"
	"net/http"
	"math/rand"
//	"strings"
	"errors"

	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/service"
	"github.com/avast/retry-go/v3"
)

type Message struct {
        NodeName string `json:"nodename"`
	NodeType string `json:"nodetype"`
        Nodes []catalog.Node `json:"nodes"`
        Services []service.Service `json:"services"`
}

func sendCatalogSync(conn *net.UDPConn, target string, catalogCopy []byte) {
	addr, err := net.ResolveUDPAddr("udp", target+":443")
        if err != nil {
                //log.Fatalln("Invalid IP address")
		return
        }
	encMessage,_ := security.EncData(("gossip -- " + string(catalogCopy)))
	//fmt.Printf("LEN OF CATALOG: %v\n", len([]byte(encMessage)))
	_,err = conn.WriteTo([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("SCS: Couldn't send response %v", err)
	}
}

func sendHealthResp(conn *net.UDPConn, addr *net.UDPAddr) {
	dt := time.Now().UTC()
	encMessage,_ := security.EncData("OK -- " + dt.String())
	_,err := conn.WriteTo([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("SHR: Couldn't send response %v", err)
	}

}

func sendHealthProbe(conn *net.UDPConn, target string) bool {
	addr, err := net.ResolveUDPAddr("udp", target+":443")
	if err != nil {
		//log.Fatalln("Invalid IP address")
		return false
	}
	encMessage,_ := security.EncData(("Health Check -- REQ -- " + target))
	_, err = conn.WriteTo([]byte(encMessage), addr)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4096)
	for {
		//n,err := conn.Read(buf)
		_,err := conn.Read(buf)
		if err != nil {
			/*
			if e, ok := err.(net.Error); !ok || !e.Timeout() {
				conn.Close()
				return false
			}
			*/
			//conn.Close()
			return false
		} else {
			// Process response
			/*
			resp := string(security.DecData(string(buf[:n])))
			if strings.Contains(resp, "OK") {
				fmt.Println("valid health resp")
				fmt.Println("THEIR: ", resp[3:])
				fmt.Println("MY DT: ", time.Now().UTC())
			}
			*/
			//conn.Close()
			return true
		}
	}
}

func CheckHealth(conn *net.UDPConn) {
	time.Sleep(200 * time.Millisecond)
	for {
		//Pull current catalog
		hname, err := os.Hostname()
		if err != nil {
			log.Fatalln("Unable to get hostname")
		}
		var body []byte
		err = retry.Do(
			func() error {
				resp, err := security.TLSGetReq(hname, "/anvil/catalog", "", "")
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
		)
		//resp, err := security.TLSGetReq(hname, "/anvil/catalog", "")
		//resp, err := http.Get("http://" + hname + ":443/anvil/catalog")
		/*
		if err != nil {
			log.Fatalln("Unable to get response")
		}

		body, err := ioutil.ReadAll(resp.Body)
		*/
		var receivedStuff Message

		err = json.Unmarshal(body, &receivedStuff)
		if err != nil {
			//log.Fatalln("Unable to decode JSON")
			time.Sleep(200 * time.Millisecond)
			//time.Sleep(5*time.Second)
		}

		if (len(receivedStuff.Nodes) > 0) {
			target := rand.Intn(len(receivedStuff.Nodes))
			if(receivedStuff.Nodes[target].Name != hname) {
				status := sendHealthProbe(conn, receivedStuff.Nodes[target].Name)
				if (status != true) {
					catalog.Deregister(receivedStuff.Nodes[target].Name)
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
		//time.Sleep(5 * time.Second)
	}
}

func PropagateCatalog(conn *net.UDPConn) {
	time.Sleep(1000 * time.Millisecond)
	//time.Sleep(5 * time.Second)
	for {
		//Pull current catalog
		hname, err := os.Hostname()
		if err != nil {
			log.Fatalln("Unable to get hostname")
		}
		var body []byte
		err = retry.Do(
			func() error {
				resp, err := security.TLSGetReq(hname, "/anvil/catalog", "", "")
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
		)
		//resp, err := security.TLSGetReq(hname, "/anvil/catalog", "")
		//resp, err := http.Get("http://" + hname + ":443/anvil/catalog")
		/*
		if err != nil {
			log.Fatalln("Unable to get response")
		}
		body, err := ioutil.ReadAll(resp.Body)
		*/
		var receivedStuff Message
		err = json.Unmarshal(body, &receivedStuff)
		if err != nil {
			//log.Fatalln("Unable to decode JSON")
			time.Sleep(1000 * time.Millisecond)
			//time.Sleep(5*time.Second)
			continue
		}
		target := rand.Intn(len(receivedStuff.Nodes))
		if(receivedStuff.Nodes[target].Name != hname) {
			var jsonData []byte
			//Pass your catalog contents back to joiner
			jsonData, err = json.Marshal(receivedStuff)
			if err != nil {
				//log.Fatalln("Unable to marshal JSON")
				time.Sleep(1000 * time.Millisecond)
				//time.Sleep(5*time.Second)
				continue
			}
			sendCatalogSync(conn, receivedStuff.Nodes[target].Name, jsonData)
		}
		time.Sleep(1000 * time.Millisecond)
		//time.Sleep(5 * time.Second)
	}
}
