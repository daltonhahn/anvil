package network

import (
	"os"
	"os/exec"
	"strings"
	"log"
	"net"
	"reflect"
	"bufio"

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
		return false
	}
	return true
}

func SaveIpTables() bool {
	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
		return false
	}

	chains, err := ipt.ListChains("nat")
	if err != nil {
		logging.InfoLogger.Println("NAT chain does not exist in iptables")
		return false
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
        log.Fatalln(err)
		return false
    }
    defer outfile.Close()
	cmd := exec.Command(cmd_path.String() +"-save")
    cmd.Stdout = outfile
    err = cmd.Start(); if err != nil {
        log.Fatalln(err)
		return false
    }
    cmd.Wait()

	logging.InfoLogger.Printf(logging.Spacer())
	return true
}

func clearTables() {
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
		_ = ipt.ClearAndDeleteChain("nat", c)
	}
}

func RestoreIpTables() {
	clearTables()
	logging.InfoLogger.Printf("Restoring previous IPTables rules from saved rules file at: .iptables-rules\n")

	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
	}
	obj_ref := reflect.ValueOf(*ipt)
	cmd_path := obj_ref.FieldByName("path")
	exec.Command(cmd_path.String() +"-restore", "<", "./.tables-rules").Output()

	logging.InfoLogger.Printf(logging.Spacer())
}

func MakeIpTables() bool {
	clearTables()

	ipt, err := iptables.New()
	if err != nil {
		log.Fatalln("IPTables not available, try running as root, or install the iptables utility")
		return false
	}
	err = ipt.ClearChain("nat", "PROXY_INIT_REDIRECT")
	if err != nil {
		logging.InfoLogger.Printf("ClearChain (of empty) failed: %v", err)
		return false
	}
	err = ipt.ClearChain("nat", "PROXY_INIT_OUTPUT")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "1:21", "-j", "REDIRECT", "--to-port", "443")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "445:8079", "-j", "REDIRECT", "--to-port", "443")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "PROXY_INIT_REDIRECT", "-p", "tcp", "--dport", "8081:65389", "-j", "REDIRECT", "--to-port", "443")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "PROXY_INIT_REDIRECT", "-p", "udp", "-j", "REDIRECT", "--to-port", "443")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "PREROUTING", "-j", "PROXY_INIT_REDIRECT")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}

	net_int, err := net.Interfaces()
	if err != nil {
		logging.InfoLogger.Printf("Could not get interfaces")
		return false
	}

	var outboundIFace string
	for _,i := range net_int {
		if (len(i.Flags.String()) > 1 && !strings.Contains(i.Flags.String(), "loopback")) {
			outboundIFace = i.Name
		}

	}
	err = ipt.Append("nat", "OUTPUT", "-p", "tcp", "-o", outboundIFace, "--dport", "80", "-j", "REDIRECT", "--to-port", "444")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	err = ipt.Append("nat", "OUTPUT", "-p", "tcp", "-o", outboundIFace, "--dport", "80", "-j", "PROXY_INIT_REDIRECT")
	if err != nil {
		logging.InfoLogger.Printf("Append failed: %v", err)
		return false
	}
	return true
}


func SetHosts(hostName string) {
	input, err := os.Open("/etc/hosts")
	if err != nil {
			log.Fatalln(err)
	}
    var lines []string
    scanner := bufio.NewScanner(input)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
	fileCopy, err := os.OpenFile("/etc/hosts.orig", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.InfoLogger.Printf("-- Can't open /etc/hosts.orig for writing -- %v\n", err)
		log.Fatalln(err)
	}
	defer fileCopy.Close()
	copyWriter := bufio.NewWriter(fileCopy)
	for _, l := range lines {
		_,_ = copyWriter.WriteString(l + "\n")
	}
	copyWriter.Flush()
	fileCopy.Close()


	fileMod, err := os.OpenFile("/etc/hosts.temp", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.InfoLogger.Printf("-- Can't open /etc/hosts.temp for writing -- %v\n", err)
		log.Fatalln(err)
	}
	defer fileMod.Close()
	modifiedWriter := bufio.NewWriter(fileMod)
	for i, line := range lines {
		if strings.Contains(line, "127.0.0.1") {
			lines[i] = "127.0.0.1\tlocalhost " + hostName
		}
		if strings.Contains(line, "127.0.1.1") {
			lines[i] = "127.0.1.1\tlocalhost " + hostName
		}
	}
	for _,l := range lines {
		_,_ = modifiedWriter.WriteString(l + "\n")
	}
	modifiedWriter.Flush()
	fileMod.Close()

	exec.Command("/bin/cp", "-f", "/etc/hosts.temp", "/etc/hosts").Output()
}
