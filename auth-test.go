package main

import (
	"io/ioutil"
	"log"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"crypto/x509"
	"crypto/tls"
)


func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", Index)
    log.Fatal(http.ListenAndServe(":8882", r))
}

func Index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello from my sample_webapp\n")

	fmt.Println("Making request to DB service")

	caCertPaths := []string{"/home/anvil/anvil/config/certs/0/server2.crt", "/home/anvil/anvil/config/certs/server1.crt", "/home/anvil/anvil/config/certs/server3.crt"}
        caCertPool := x509.NewCertPool()
        for _, fp := range caCertPaths {
                caCert, err := ioutil.ReadFile(fp)
                if err != nil {
                        log.Println("Read file error in POST REQ SVC")
                }
                caCertPool.AppendCertsFromPEM(caCert)
        }
        cert,err := tls.LoadX509KeyPair("/home/anvil/anvil/config/certs/0/server2.crt", "/home/anvil/anvil/config/certs/0/server2.key")
        if err != nil {
		log.Fatalln(err)
        }

        client := &http.Client{
                Transport: &http.Transport{
                        TLSClientConfig: &tls.Config{
                                RootCAs:      caCertPool,
                                Certificates: []tls.Certificate{cert},
                        },
                },
        }

	req, err = http.NewRequest("GET", ("http://172.18.0.4/outbound/auth/service/db"), nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln("Unable to get response")
	}
	body, err := ioutil.ReadAll(resp.Body)

	fmt.Println("Received: ", string(body))
}
