package catalog

import (
	"fmt"

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
