package catalog

import (
	"errors"
	"fmt"
	"net"

	//"github.com/daltonhahn/anvil/envoy"
)

var AnvilCatalog Catalog

type Node struct {
	Name	string
	Address	string
	Type	string
}

type Service struct {
	Name	string
	Address	string
	Port	int64
}

type Catalog struct {
	Nodes		[]Node
	Services	[]Service
}

func (catalog *Catalog) AddService(newSvc Service) []Service {
	catalog.Services = append(catalog.Services, newSvc)
	return catalog.Services
}

func (catalog *Catalog) AddNode(newNode Node) []Node {
	catalog.Nodes = append(catalog.Nodes, newNode)
	return catalog.Nodes
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

func (catalog *Catalog) RemoveService(targetName string, targetAddr string) []Service {
	filteredServices := catalog.Services[:0]
	for _,svc := range catalog.Services {
		if svc.Name != targetName || svc.Address != targetAddr {
			filteredServices = append(filteredServices, svc)
		}
	}
	return filteredServices
}

func Register(nodeName string, svcList []Service, nodeType string) {
	addr, err := net.LookupIP(nodeName)
	if err != nil {
		fmt.Println("Lookup failed")
	}
	AnvilCatalog.Nodes = AnvilCatalog.AddNode(Node{Name: nodeName, Address: addr[0].String(), Type: nodeType})
	for _, ele := range svcList {
		AnvilCatalog.Services = AnvilCatalog.AddService(Service{ele.Name, addr[0].String(), ele.Port})
	}
}

func Deregister(nodeName string) {
	for _, ele := range AnvilCatalog.Nodes {
		if ele.Name == nodeName {
			AnvilCatalog.Nodes = AnvilCatalog.RemoveNode(nodeName)
			for _, svc := range AnvilCatalog.Services {
				addr, _ := net.LookupIP(nodeName)
				if svc.Address == addr[0].String() {
					AnvilCatalog.Services = AnvilCatalog.RemoveService(svc.Name, svc.Address)
				}
			}
		}
	}
}

func GetCatalog() *Catalog {
	return &AnvilCatalog
}

func (catalog *Catalog) PrintNodes() {
	fmt.Println("\t---- Nodes within Anvil ----")
	for _, ele := range AnvilCatalog.Nodes {
		fmt.Println("\t",ele)
	}

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
func (catalog *Catalog) GetServices() ([]Service) {
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
