package gossip

import (
	"strings"
	"time"
	"net"
	"net/http"
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"bufio"
	"github.com/google/gopacket"
	layers "github.com/google/gopacket/layers"

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
	_,err := conn.WriteToUDP([]byte("OK " + dt.String()), addr)
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
	}
}

func HandleUDP(p []byte, ser *net.UDPConn) {
	// WHEN READING UDP MESSAGE FROM SOCKET, DECRYPT WITH KEY -- LATER
	for {
		n,remoteaddr,err := ser.ReadFromUDP(p)

		message := string(p[:n])

		// Parse content received (p) to determine health check vs. gossip
		if message == "health" {
			sendHealthResp(ser, remoteaddr)
		} else if (len(message) > 6 && message[:6] == "gossip") {
			if err !=  nil {
				fmt.Printf("Some error  %v", err)
				continue
			}
			sendResponse(ser, remoteaddr)
		} else {
			//Check if this is a valid DNS file
			packet := gopacket.NewPacket(p, layers.LayerTypeDNS, gopacket.Default)
			dnsPacket := packet.Layer(layers.LayerTypeDNS)
			tcp,valid := dnsPacket.(*layers.DNS)
			if valid != true {
				continue
			} else {
				serveDNS(ser, remoteaddr, tcp)
			}
		}
	}
}

func serveDNS(u *net.UDPConn, clientAddr net.Addr, request *layers.DNS) {
	replyMess := request
	var dnsAnswer layers.DNSResourceRecord
	dnsAnswer.Type = layers.DNSTypeA
	var ip string
	var err error
	// REPLACE WITH ON-THE-FLY RECORD LOOKUP
	ip,err = catalog.LookupDNS(string(request.Questions[0].Name[:strings.IndexByte(string(request.Questions[0].Name), '.')]))

	if err != nil {
		fmt.Println(err)
		//Todo: Log no data present for the IP and handle:todo
	}
	a, _, _ := net.ParseCIDR(ip + "/24")
	dnsAnswer.Type = layers.DNSTypeA
	dnsAnswer.IP = a
	dnsAnswer.Name = []byte(request.Questions[0].Name)
	dnsAnswer.Class = layers.DNSClassIN
	replyMess.QR = true
	replyMess.ANCount = 1
	replyMess.OpCode = layers.DNSOpCodeNotify
	replyMess.AA = true
	replyMess.Answers = append(replyMess.Answers, dnsAnswer)
	replyMess.ResponseCode = layers.DNSResponseCodeNoErr
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{} // See SerializeOptions for more details.
	err = replyMess.SerializeTo(buf, opts)
	if err != nil {
		panic(err)
	}
	u.WriteTo(buf.Bytes(), clientAddr)
}
