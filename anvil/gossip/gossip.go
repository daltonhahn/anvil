package gossip

import (
	"time"
	"net"
	"fmt"
)


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
