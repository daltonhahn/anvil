package network

import (
	"os"
	"os/exec"
	"io/ioutil"
	"strings"
	"log"
	"net"
	"fmt"
	"reflect"

	"github.com/coreos/go-iptables/iptables"
	"github.com/daltonhahn/anvil/logging"
)

func CheckTables() bool {
	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
		return false
	}
	_, err = ipt.ListChains("nat")
	if err != nil {
		logging.InfoLogger.Println("NAT chain does not exist in iptables")
	}
	return true
}

func SaveIpTables() {
	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
	}

	chains, err := ipt.ListChains("nat")
	if err != nil {
		logging.InfoLogger.Println("NAT chain does not exist in iptables")
	}
	logging.InfoLogger.Printf(logging.Spacer())
	logging.InfoLogger.Printf("Available iptables chains: %v\n", chains)

	for _, c := range chains {
		rules, err := ipt.List("nat", c)
		if err != nil {
			logging.InfoLogger.Println("Unable to get rules from chain")
			break
		} else {
			for _, rule := range rules {
				logging.InfoLogger.Printf("\t%v\n", rule)
			}
		}
	}
	obj_ref := reflect.ValueOf(*ipt)
	cmd_path := obj_ref.FieldByName("path")

    // open the out file for writing
	// In the future, transition this to be within the data-dir
    outfile, err := os.Create("./.tables-rules")
    if err != nil {
        panic(err)
    }
    defer outfile.Close()
	cmd := exec.Command(cmd_path.String() +"-save")
    cmd.Stdout = outfile
    err = cmd.Start(); if err != nil {
        log.Fatalln(err)
    }
    cmd.Wait()

	logging.InfoLogger.Printf(logging.Spacer())
}

func RestoreIpTables() {
	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
	}
	err = ipt.ClearAll()
	if err != nil {
		log.Fatalln("Issue clearing current IPTables chains and rules")
	}
	err = ipt.DeleteAll()
	if err != nil {
		log.Fatalln("Issue deleting current IPTables chains and rules")
	}
	chains, err := ipt.ListChains("nat")
	if err != nil {
		logging.InfoLogger.Println("NAT table does not exist in iptables")
	}
	for _,c := range chains {
		err = ipt.ClearAndDeleteChain("nat", c)
		if err != nil {
			logging.InfoLogger.Printf("Unable to clear chain: %v\n", c)
		}
	}
	logging.InfoLogger.Printf("Restoring previous IPTables rules from saved rules file at: .iptables-rules\n")

	obj_ref := reflect.ValueOf(*ipt)
	cmd_path := obj_ref.FieldByName("path")
	exec.Command(cmd_path.String() +"-restore", "<", "./.tables-rules").Output()

	logging.InfoLogger.Printf(logging.Spacer())
}




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
