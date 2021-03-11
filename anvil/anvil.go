package anvil

import (
	"fmt"
	"time"
	"os"
	"os/exec"
	"log"
	"net"
	"net/http"
	"io/ioutil"
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"

	"github.com/daltonhahn/anvil/envoy"
	"github.com/daltonhahn/anvil/iptables"
	"github.com/daltonhahn/anvil/anvil/gossip"
)

func Join(target string) {
	fmt.Println("Node to join: ", target, "\n")
	postBody, _ := json.Marshal(map[string]string{
		"name": "testing_testing",
		"svc": "empty for now",
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("http://" + target + "/anvil/catalog/register", "application/json", responseBody)

	if err != nil {
		log.Fatalf("An error occured %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	fmt.Println(sb)

	//Trigger HTTP request to target node /catalog/register endpoint
	//If target responds with 200 OK, add target to your own catalog
}

func AnvilInit() {
        envoy.SetupEnvoy()
        cmd := &exec.Cmd {
                Path: "/usr/bin/envoy",
                Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy.yaml" },
                Stdout: os.Stdout,
                Stderr: os.Stdout,
        }
        cmd.Start()

	iptables.CleanTables()
        iptables.MakeIpTables()

        router := mux.NewRouter()
	router.HandleFunc("/catalog", GetCatalog).Methods("GET")
	router.HandleFunc("/catalog/register", RegisterNode).Methods("POST")
	router.HandleFunc("/", Index).Methods("GET")

	p := make([]byte, 2048)

	addr := net.UDPAddr{
		Port: 8080,
		IP: net.ParseIP("0.0.0.0"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}

	go gossip.HandleUDP(p, ser)

        log.Fatal(http.ListenAndServe(":8080", router))

        cmd.Wait()
}

func RegisterNode(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hit the register endpoint\n")
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome!\n")
}

func GetCatalog(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Catalog at " + dt.String() + "\n"))
}
