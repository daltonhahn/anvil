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

func Register(nodeName string, svcList []Service) {
	AnvilCatalog.Nodes = AnvilCatalog.AddNode(Node{Name: nodeName, Address: nodeName})
	for _, ele := range svcList {
		AnvilCatalog.Services = AnvilCatalog.AddService(Service{ele.Name, nodeName, ele.Port})
	}
}

func Deregister(nodeName string) {
	fmt.Println("WORKING TO REMOVE ", nodeName, " FROM CATALOG")
	//Select the node to be removed from the list of nodes

	//Loop through all services and compare the stored address with the
	//address of the node to be removed
		//If match, remove that service as well
		//Else, continue

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
		fmt.Println(ele.Name, "\t\t\t", ele.Address, "\t\t", ele.Port)
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
