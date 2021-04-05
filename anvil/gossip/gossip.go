package gossip

import (
	"time"
	"net"
	"net/http"
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"bufio"

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
	fmt.Println("Sending Resp: ", encMessage)
	_,err := conn.WriteToUDP([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}
}

func sendHealthResp(conn *net.UDPConn, addr *net.UDPAddr) {
	dt := time.Now()
	encMessage := security.EncData("OK " + dt.String())
	fmt.Println("Sending HealthResp: ", encMessage)
	_,err := conn.WriteToUDP([]byte(encMessage), addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}

}

func sendHealthProbe(target string) bool {
	p := make([]byte, 2048)
	conn, err := net.Dial("udp", target+":80")
	if err != nil {
		log.Fatalln("Unable to connect to target")
	}
	encMessage := security.EncData("health")
	fmt.Println("Sending Probe: ", encMessage)
	fmt.Fprintf(conn, string(encMessage))
	_, err = bufio.NewReader(conn).Read(p)
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

		for _, ele := range receivedStuff.Nodes {
			status := sendHealthProbe(ele.Name)

			if (status != true) {
				catalog.Deregister(ele.Name)
			}
		}
		time.Sleep(10 * time.Second)
	}
}
