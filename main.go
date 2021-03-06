package main

import (
	"time"
	"fmt"
	"os"
	"os/exec"
	"log"
	"net/http"
	"io/ioutil"
	"strconv"

	"gopkg.in/yaml.v2"
	"github.com/julienschmidt/httprouter"
)

const listener_config = `resources:
- "@type": type.googleapis.com/envoy.config.listener.v3.Listener
  name: listener_0
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 80
  filter_chains:
  - filters:
      name: envoy.filters.network.http_connection_manager
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
        stat_prefix: ingress_http
        http_filters:
        - name: envoy.filters.http.router
        route_config:
          name: local_route
          virtual_hosts:
          - name: backend
            domains: 
            - "*"
            routes:
            - match:
                prefix: "/anvil"
              route:
                prefix_rewrite: "/"
                cluster: anvil_service
`


type Service struct {
	Name	string
	Port	int64
}

type EnvoyConfig struct {
	Services	[]Service
}

func writeAnvilCluster(f_cds *os.File) *os.File {
	if _, err := f_cds.WriteString(`resources:
- "@type": type.googleapis.com/envoy.config.cluster.v3.Cluster
  name: anvil_service
  connect_timeout: 30s
  type: LOGICAL_DNS
  dns_lookup_family: V4_ONLY
  load_assignment:
    cluster_name: anvil_service
    endpoints:
    - lb_endpoints:
      - endpoint:
          address:
            socket_address:
              address: 0.0.0.0
              port_value: 8080

`); err != nil {
		log.Printf("Writing error %v", err)
	}
	return f_cds
}

func writeCDS(svc string, port int64, f_cds *os.File) *os.File {
	if _, err := f_cds.WriteString("- \"@type\": type.googleapis.com/envoy.config.cluster.v3.Cluster\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_cds.WriteString("  name: " + svc + "_service\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_cds.WriteString(`  connect_timeout: 30s
  type: LOGICAL_DNS
  dns_lookup_family: V4_ONLY
  load_assignment:
  `); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_cds.WriteString("  cluster_name: " + svc + "_service"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_cds.WriteString(`
    endpoints:
    - lb_endpoints:
      - endpoint:
          address:
            socket_address:
              address: 0.0.0.0
`); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_cds.WriteString("              port_value: " + strconv.FormatInt(port,10) + "\n\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	return f_cds
}

func writeLDS(svc string, port int64, f_lds *os.File) *os.File {
	if _, err := f_lds.WriteString("            - match:\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_lds.WriteString("                prefix: \"/service/" + svc + "\"\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_lds.WriteString("              route:\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_lds.WriteString("                cluster: " + svc + "_service\n"); err != nil {
		log.Printf("Writing error %v", err)
	}

	return f_lds
}

func setupEnvoy() {
	f_lds, err := os.OpenFile("/root/anvil/envoy/config/lds.yaml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Open file error %v", err)
	}
	defer f_lds.Close()

	f_cds, err := os.OpenFile("/root/anvil/envoy/config/cds.yaml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Open file error %v", err)
	}
	defer f_cds.Close()

	f_cds = writeAnvilCluster(f_cds)
	c, err := readEnvoyConfig()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := f_lds.WriteString(listener_config); err != nil {
		log.Printf("Writing error %v", err)
	}
	for _,ele := range c.Services {
		fmt.Printf("%s:%d\n", ele.Name, ele.Port)
		f_cds = writeCDS(ele.Name, ele.Port, f_cds)
		f_lds = writeLDS(ele.Name, ele.Port, f_lds)
	}
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func getCatalog(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Catalog at " + dt.String() + "\n"))
}

func getNodeServices(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node_name := params.ByName("node")
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Services for Node: " + node_name + "\nCurrent Time: " + dt.String() + "\n"))
}

func makeIpTables() {
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-N", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "tcp", "-j",
		"REDIRECT", "--to-port", "80").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "udp", "-j",
		"REDIRECT", "--to-port", "80").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PREROUTING", "-j", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-N", "PROXY_INIT_OUTPUT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_OUTPUT", "-o", "lo", "!", "-d",
		"127.0.0.1/32", "-j", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "OUTPUT", "-j", "PROXY_INIT_OUTPUT").Output()
}

func readEnvoyConfig() (*EnvoyConfig, error) {
	yamlFile, err := ioutil.ReadFile("/root/anvil/envoy/config/sample-svc.yaml")
	if err != nil {
		log.Printf("Read file error #%v", err)
	}
	c := &EnvoyConfig{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c, nil
}

func main() {
	setupEnvoy()
	cmd := &exec.Cmd {
		Path: "/usr/bin/envoy",
		Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy.yaml" },
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	cmd.Start()

	makeIpTables()

	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/catalog", getCatalog)
	router.GET("/catalog/:node", getNodeServices)

	log.Fatal(http.ListenAndServe(":8080", router))
	cmd.Wait()
}
