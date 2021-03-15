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

	"github.com/daltonhahn/anvil/catalog"
)

type Message struct {
        NodeName string `json:"nodename"`
        Nodes []catalog.Node `json:"nodes"`
        Services []catalog.Service `json:"services"`
}

func sendResponse(conn *net.UDPConn, addr *net.UDPAddr) {
	// BEFORE SENDING A RESPONSE, ENCRYPT WITH KEY -- LATER

	_,err := conn.WriteToUDP([]byte("TRASHY GOSSIP\n"), addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}
}

func sendHealthResp(conn *net.UDPConn, addr *net.UDPAddr) {
	// BEFORE SENDING A RESPONSE, ENCRYPT WITH KEY -- LATER

	dt := time.Now()
	_,err := conn.WriteToUDP([]byte("OK " + dt.String() + "\n"), addr)
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
	fmt.Fprintf(conn, "health")
	_, err = bufio.NewReader(conn).Read(p)
	if err != nil {
		conn.Close()
		return false
	} else {
		fmt.Printf("%s\n", p)
		conn.Close()
		return true
	}
}

func CheckHealth(conn *net.UDPConn) {
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
		fmt.Println("Restarting loop")
	}

	//Get random subset of nodes

	//Send health check request to nodes

	//If response missing, print error and update catalog

	//Sleep for 10 seconds
}

func HandleUDP(p []byte, ser *net.UDPConn) {
	// WHEN READING UDP MESSAGE FROM SOCKET, DECRYPT WITH KEY -- LATER
	for {
		n,remoteaddr,err := ser.ReadFromUDP(p)

		test := "health"
		// Parse content received (p) to determine health check vs. gossip
		if string(p[:n]) == test {
			fmt.Printf("Health check received from %v\n", remoteaddr)
			sendHealthResp(ser, remoteaddr)
		} else {
			fmt.Printf("Gossip received from: %v -- %s \n", remoteaddr, p)
			if err !=  nil {
				fmt.Printf("Some error  %v", err)
				continue
			}
			sendResponse(ser, remoteaddr)
		}
	}
}
