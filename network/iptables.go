package network

import (
	"os/exec"
	"io/ioutil"
	"strings"
	"log"
	"net"
	"fmt"
)

func MakeIpTables() {
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-N", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "1:21", "-j",
		"REDIRECT", "--to-port", "443").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "23:442", "-j",
		"REDIRECT", "--to-port", "443").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "445:8079", "-j",
		"REDIRECT", "--to-port", "443").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "8081:65389", "-j",
		"REDIRECT", "--to-port", "443").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PROXY_INIT_REDIRECT", "-p", "udp", "-j",
		"REDIRECT", "--to-port", "443").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "PREROUTING", "-j", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-N", "PROXY_INIT_OUTPUT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "-o", "ens192", "--dport",
		"80", "-j", "REDIRECT", "--to-port", "444").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "-o", "ens192", "--dport",
		"80", "-j", "PROXY_INIT_REDIRECT").Output()
}

func CleanTables() {
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PREROUTING").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PROXY_INIT_REDIRECT_OUTBOUND").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PROXY_INIT_OUTPUT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-D", "OUTPUT", "3").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-D", "OUTPUT", "2").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-D", "OUTPUT", "1").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "--delete-chain", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "--delete-chain", "PROXY_INIT_OUTPUT").Output()
}

func SetHosts(hostName string) {
        input, err := ioutil.ReadFile("/etc/hosts")
        if err != nil {
                log.Fatalln(err)
        }

        lines := strings.Split(string(input), "\n")

        for i, line := range lines {
                if strings.Contains(line, "127.0.0.1") {
                        lines[i] = "127.0.0.1\tlocalhost " + hostName
                }
		if strings.Contains(line, "127.0.1.1") {
			fmt.Println("FOUND MY LINE")
			lines[i] = GetOutboundIP().String() + "\t" + hostName
		}
        }
        output := strings.Join(lines, "\n")
        err = ioutil.WriteFile("/etc/hosts.temp", []byte(output), 0644)
        if err != nil {
                log.Fatalln(err)
        }
	exec.Command("/bin/cp", "-f", "/etc/hosts.temp", "/etc/hosts").Output()
}

func GetOutboundIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP
}
