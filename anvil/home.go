package main

import (
	"fmt"
	"os"
	"os/exec"
	"log"
	"net/http"
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"github.com/julienschmidt/httprouter"
)

type Service struct {
	Name	string
	Port	int64
}

type EnvoyConfig struct {
	Services	[]Service
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
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
	cmd := &exec.Cmd {
		Path: "/usr/bin/envoy",
		Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy-demo.yaml" },
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	cmd.Start()

	makeIpTables()

	c, err := readEnvoyConfig()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d\n", c.Services[0].Port)

	router := httprouter.New()
	router.GET("/", Index)

	log.Fatal(http.ListenAndServe(":8080", router))
	cmd.Wait()
}
