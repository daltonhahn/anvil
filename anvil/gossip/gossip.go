package gossip

import (
	"time"
	"net"
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/catalog"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []catalog.Service `json:"services"`
}

func sendResponse(conn *net.UDPConn, addr *net.UDPAddr) {
	encMessage := security.EncData("Trashy Gossip\n")
	_,err := conn.WriteToUDP([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}
}

func sendHealthResp(conn *net.UDPConn, addr *net.UDPAddr) {
	dt := time.Now()
	encMessage := security.EncData("OK " + dt.String())
	_,err := conn.WriteToUDP([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}

}

func sendHealthProbe(target string) bool {
	_, err := net.ResolveUDPAddr("udp4", target+":443")
	if err != nil {
		log.Fatalln("Invalid IP address")
	}
	conn, err := net.Dial("udp", target+":443")
	if err != nil {
		log.Fatalln("Unable to connect to target")
	}
	encMessage := security.EncData(("Health Check -- REQ -- " + target))
	_, err = conn.Write([]byte(encMessage))
	if err != nil {
		conn.Close()
		return false
	} else {
		conn.Close()
		return true
	}
}

func CheckHealth() {
	time.Sleep(10 * time.Second)
	for {
		//Pull current catalog
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

		for _, ele := range receivedStuff.Nodes {
			if (ele.Name != hname) {
				status := sendHealthProbe(ele.Name)
				if (status != true) {
					catalog.Deregister(ele.Name)
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}
