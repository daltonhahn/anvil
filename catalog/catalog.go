package catalog

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/daltonhahn/anvil/raft"
	"github.com/daltonhahn/anvil/service"
)

var AnvilCatalog Catalog

type Node struct {
	Name	string
	Address	string
	Type	string
}

type Catalog struct {
	Nodes		[]Node
	Services	[]service.Service
}

func (catalog *Catalog) AddService(newSvc service.Service) []service.Service {
	for ind, ele := range catalog.Services {
		if ele.Name == newSvc.Name && ele.Address == newSvc.Address {
			 catalog.Services[ind] = newSvc
			 return catalog.Services
		 }
	 }
	catalog.Services = append(catalog.Services, newSvc)
	return catalog.Services
}

func (catalog *Catalog) AddNode(newNode Node) []Node {
	for ind, ele := range catalog.Nodes {
		if ele.Name == newNode.Name {
			catalog.Nodes[ind] = newNode
			return catalog.Nodes
		}
	}
	catalog.Nodes = append(catalog.Nodes, newNode)
	return catalog.Nodes
}

func AddPeer(peerList []string, targetName string) []string {
	for _, ele := range peerList {
		if ele == targetName {
			return peerList
		}
	}
	peerList = append(peerList, targetName)
	return peerList
}

func (catalog *Catalog) RemoveNode(target string) []Node {
	filteredNodes := catalog.Nodes[:0]
	for _,node := range catalog.Nodes {
		if node.Name != target {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

func (catalog *Catalog) RemoveService(targetName string, targetAddr string) []service.Service {
	filteredServices := catalog.Services[:0]
	for _,svc := range catalog.Services {
		if svc.Name != targetName || svc.Address != targetAddr {
			filteredServices = append(filteredServices, svc)
		}
	}
	return filteredServices
}

func RemovePeer(peerList []string, targetName string) []string {
	filteredPeers := peerList[:0]
	for _,peer := range peerList {
		if peer != targetName {
			filteredPeers = append(filteredPeers,peer)
		}
	}
	return filteredPeers
}

func UpdateNodeTypes(newLeader string) {
	for ind, ele := range AnvilCatalog.Nodes {
		if ele.Name == newLeader {
			AnvilCatalog.Nodes[ind].Type = "leader"
		} else if ele.Name != newLeader && ele.Type != "client" {
			AnvilCatalog.Nodes[ind].Type = "server"
		}
	}
}

func Register(nodeName string, svcList []service.Service, nodeType string) {
	addr, err := net.LookupIP(nodeName)
	if err != nil {
		fmt.Println("Lookup failed")
	}
	AnvilCatalog.Nodes = AnvilCatalog.AddNode(Node{Name: nodeName, Address: addr[0].String(), Type: nodeType})
	hname, err := os.Hostname()
	hostIPs, _ := net.LookupIP(hname)
	var targIP net.IP
	if hostIPs[0].Equal(net.ParseIP("127.0.0.1")) {
		targIP = hostIPs[1]
	} else {
		targIP = hostIPs[0]
	}
	if (nodeType == "server" || nodeType == "leader") && nodeName != hname && nodeName != targIP.String() {
		raft.CM.PeerIds = AddPeer(raft.CM.PeerIds, nodeName)//addr[0].String())
	}
	for _, ele := range svcList {
		AnvilCatalog.Services = AnvilCatalog.AddService(service.Service{ele.Name, addr[0].String(), ele.Port})
	}
}

func Deregister(nodeName string) {
	for _, ele := range AnvilCatalog.Nodes {
		if ele.Name == nodeName {
			AnvilCatalog.Nodes = AnvilCatalog.RemoveNode(nodeName)
			for _, svc := range AnvilCatalog.Services {
				addr, err := net.LookupIP(nodeName)
				if err == nil && addr != nil {
					if svc.Address == addr[0].String() {
						AnvilCatalog.Services = AnvilCatalog.RemoveService(svc.Name, svc.Address)
					}
				}
			}
		}
	}
	raft.CM.PeerIds = RemovePeer(raft.CM.PeerIds, nodeName)
}

func GetCatalog() *Catalog {
	return &AnvilCatalog
}

func (catalog *Catalog) GetNodeType(nodeName string) (string) {
	for _, ele := range AnvilCatalog.Nodes {
		if ele.Name == nodeName {
			return ele.Type
		}
	}
	return ""
}

func (catalog *Catalog) PrintNodes() {
	fmt.Println("\t---- Nodes within Anvil ----")
	for _, ele := range AnvilCatalog.Nodes {
		fmt.Println("\t",ele)
	}
	fmt.Printf("\t --- Number of nodes: %v ---\n", len(AnvilCatalog.Nodes))
}

func (catalog *Catalog) PrintServices() {
	fmt.Println("\t---- Services within Anvil ----")
	fmt.Println("Service Name\t\tAddress\t\tPort")
	for _, ele := range AnvilCatalog.Services {
		fmt.Println(ele.Name, "\t\t", ele.Address, "\t\t", ele.Port)
	}
}

func (catalog *Catalog) GetNodes() ([]Node) {
	return AnvilCatalog.Nodes
}

func (catalog *Catalog) GetSvcHost(addr string) (string) {
	for _,ele := range AnvilCatalog.Nodes {
		if ele.Address == addr {
			return ele.Name
		}
	}
	dns, err := net.LookupAddr(addr)
	if err != nil || dns == nil {
		return ""
	} else {
		return strings.Split(dns[0],".")[0]
	}
	return ""
}

func (catalog *Catalog) GetSvcPort(svc string) (int64) {
	for _,ele := range AnvilCatalog.Services {
		if ele.Name == svc {
			return ele.Port
		}
	}
	return -1
}

func (catalog *Catalog) GetClients() ([]string) {
	var clientList []string
	for _,ele := range AnvilCatalog.Nodes {
		if ele.Type == "client" {
			clientList = append(clientList, ele.Name)
		}
	}
	return clientList
}

func (catalog *Catalog) GetQuorumMem() (string) {
	for _,ele := range AnvilCatalog.Nodes {
		if ele.Type == "leader" || ele.Type == "server" {
			return ele.Name
		}
	}
	return ""
}

func (catalog *Catalog) GetPeers() ([]string) {
	retList := []string{}
	for _,ele := range AnvilCatalog.Nodes {
		if ele.Type == "leader" || ele.Type == "server" {
			retList = append(retList, ele.Name)
		}
	}
	return retList
}

func (catalog *Catalog) GetLeader() (string) {
	for _,ele := range AnvilCatalog.Nodes {
		if ele.Type == "leader" {
			return ele.Address
		}
	}
	return ""
}

func (catalog *Catalog) GetServices() ([]service.Service) {
	return AnvilCatalog.Services
}

func LookupDNS(svcName string) (string, error) {
	for _, ele := range AnvilCatalog.Services {
		if ele.Name == svcName {
			addr, err := net.LookupIP(ele.Address)
			if err != nil {
				fmt.Println("Lookup failed")
			}
			return addr[0].String(),nil
		}
	}
	return "",errors.New("Not Found")
}
