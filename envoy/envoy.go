package envoy

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"os"
	"log"
	"strconv"

	"github.com/daltonhahn/anvil/catalog"
)

const listener_config = `resources:
- "@type": type.googleapis.com/envoy.config.listener.v3.Listener
  name: listener_0
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 443
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
                prefix: "/anvil/"
              route:
                prefix_rewrite: "/"
                cluster: anvil_service
            - match:
                prefix: "/anvil"
              route:
                prefix_rewrite: "/"
                cluster: anvil_service
`
const tls_config =
`    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
        require_client_certificate: true
        common_tls_context:
          validation_context:
            trusted_ca:
              filename: /root/anvil/config/certs/ca.crt
          tls_certificates:
          - certificate_chain:
              filename: /root/anvil/config/certs/server1.crt
            private_key:
              filename: /root/anvil/config/certs/server1.key
`

const gossip_config = `
- "@type": type.googleapis.com/envoy.config.listener.v3.Listener
  name: listener_1
  reuse_port: true
  address:
      socket_address:
        protocol: UDP
        address: 0.0.0.0 
        port_value: 443
  listener_filters:
    - name: envoy.filters.udp_listener.udp_proxy
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.filters.udp.udp_proxy.v3.UdpProxyConfig
        stat_prefix: gossip
        cluster: anvil_gossip
`

var S_list = new(EnvoyConfig)

type EnvoyConfig struct {
	Services	[]catalog.Service
}

func SetServiceList() (EnvoyConfig) {
	S_list, err := readEnvoyConfig()
	if err != nil {
		log.Fatal(err)
	}
	return *S_list
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

- "@type": type.googleapis.com/envoy.config.cluster.v3.Cluster
  name: anvil_gossip
  connect_timeout: 0.25s
  type: LOGICAL_DNS
  dns_lookup_family: V4_ONLY
  load_assignment:
    cluster_name: anvil_gossip
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
	if _, err := f_lds.WriteString("                prefix_rewrite: \"/\"\n"); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_lds.WriteString("                cluster: " + svc + "_service\n"); err != nil {
		log.Printf("Writing error %v", err)
	}

	return f_lds
}

func SetupEnvoy() {
        CleanTables()
        MakeIpTables()

	f_lds, err := os.OpenFile("/root/anvil/envoy/config/lds.yaml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("Open file error %v", err)
	}
	defer f_lds.Close()

	f_cds, err := os.OpenFile("/root/anvil/envoy/config/cds.yaml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("Open file error %v", err)
	}
	defer f_cds.Close()

	f_cds = writeAnvilCluster(f_cds)
	S_list := SetServiceList()

	if _, err := f_lds.WriteString(listener_config); err != nil {
		log.Printf("Writing error %v", err)
	}
	for _,ele := range S_list.Services {
		f_cds = writeCDS(ele.Name, ele.Port, f_cds)
		f_lds = writeLDS(ele.Name, ele.Port, f_lds)
	}
	if _, err := f_lds.WriteString(tls_config); err != nil {
		log.Printf("Writing error %v", err)
	}
	if _, err := f_lds.WriteString(gossip_config); err != nil {
		log.Printf("Writing error %v", err)
	}
}

func readEnvoyConfig() (*EnvoyConfig, error) {
	yamlFile, err := ioutil.ReadFile("/root/anvil/envoy/config/sample-svc.yaml")
	if err != nil {
		log.Printf("Read file error #%v", err)
	}
	err = yaml.Unmarshal(yamlFile, S_list)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return S_list, nil
}
