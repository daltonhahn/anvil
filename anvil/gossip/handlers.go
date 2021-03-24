package gossip

import (
	"net"
	"fmt"
	"strings"
	"github.com/google/gopacket"
	layers "github.com/google/gopacket/layers"

	"github.com/daltonhahn/anvil/catalog"
)

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
