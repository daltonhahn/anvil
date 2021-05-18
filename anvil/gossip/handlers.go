package gossip

import (
	"net"
	"fmt"
	"strings"
	"log"
//	"net/http"
	"os"
	"bytes"
	"encoding/json"
	"github.com/google/gopacket"
	layers "github.com/google/gopacket/layers"

	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/security"
)

func HandleUDP(p []byte, ser *net.UDPConn) {
	for {
		n,remoteaddr,err := ser.ReadFromUDP(p)

		message := string(p[:n])
		decMessage,err := security.DecData(message)

		if err != nil {
			//Check if this is a valid DNS file
			packet := gopacket.NewPacket(p, layers.LayerTypeDNS, gopacket.Default)
			dnsPacket := packet.Layer(layers.LayerTypeDNS)
			tcp,valid := dnsPacket.(*layers.DNS)
			if valid != true {
				continue
			} else {
				serveDNS(ser, remoteaddr, tcp)
			}
		} else {
			if strings.Contains(string(decMessage), "Health Check -- REQ --") {
				sendHealthResp(ser, remoteaddr)
			} else if (len(decMessage) > 6 && string(decMessage)[:6] == "gossip") {
				if err !=  nil {
					fmt.Printf("Some error  %v", err)
					continue
				}

				var receivedStuff Message
				err = json.Unmarshal(decMessage[10:], &receivedStuff)
				if err != nil {
					log.Fatalln("Unable to decode JSON")
				}
				var tempCatalog catalog.Catalog
				for _, ele := range receivedStuff.Nodes {
					tempCatalog.AddNode(ele)
					for _, svc := range receivedStuff.Services {
						if (ele.Address == svc.Address) {
							tempCatalog.AddService(svc)
						}
					}
					var localPost Message
					localPost.NodeName = ele.Name
					localPost.Services = tempCatalog.Services
					localPost.NodeType = ele.Type
					postBody, _ := json.Marshal(localPost)
					responseBody := bytes.NewBuffer(postBody)
					// Marshal the struct into a postable message
					hname, err := os.Hostname()
					if err != nil {
						log.Fatalln("Unable to get hostname")
					}
					//http.Post("http://"+hname+":443/anvil/catalog/register", "application/json", responseBody)
					security.TLSPostReq(hname, "/anvil/catalog/register", "", "application/json", responseBody)
					tempCatalog = catalog.Catalog{}
				}
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
	ip,err = catalog.LookupDNS(string(request.Questions[0].Name[:strings.IndexByte(string(request.Questions[0].Name), '.')]))

	if err != nil {
		fmt.Println(err)
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
