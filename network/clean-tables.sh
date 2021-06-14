#!/bin/bash

iptables -t nat -F PREROUTING
iptables -t nat -F PROXY_INIT_REDIRECT
iptables -t nat -F PROXY_INIT_REDIRECT_OUTBOUND
iptables -t nat -F PROXY_INIT_OUTPUT
iptables -t nat -D OUTPUT 3 
iptables -t nat -D OUTPUT 2

iptables -t nat --delete-chain PROXY_INIT_REDIRECT
iptables -t nat --delete-chain PROXY_INIT_OUTPUT
iptables -t nat --delete-chain PROXY_INIT_REDIRECT_OUTBOUND
