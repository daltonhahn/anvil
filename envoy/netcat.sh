#!/bin/bash

#read -p "Port num: " port

while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo -e `date`; } | nc -l 444 -q 1; done
