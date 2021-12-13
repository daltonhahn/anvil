#!/bin/bash

curl http://172.18.0.4/outbound/auth/service/db/ --cacert config/certs/0/server2.crt --cert config/certs/0/server2.crt --key config/certs/0/server2.key --insecure
