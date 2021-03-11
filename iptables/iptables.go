package iptables

import (
	"os/exec"
)

func MakeIpTables() {
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

func CleanTables() {
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PREROUTING").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-F", "PROXY_INIT_OUTPUT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "-D", "OUTPUT", "2").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "--delete-chain", "PROXY_INIT_REDIRECT").Output()
	exec.Command("/usr/sbin/iptables", "-t", "nat", "--delete-chain", "PROXY_INIT_OUTPUT").Output()
}
