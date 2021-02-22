package main

import (
	"fmt"
	"os"
	"os/exec"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func main() {
	cmd := &exec.Cmd {
		Path: "/usr/bin/envoy",
		Args: []string{"/usr/bin/envoy", "-c", "/root/anvil/envoy/envoy-demo.yaml" },
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	cmd.Start();

	router := httprouter.New()
	router.GET("/", Index)

	log.Fatal(http.ListenAndServe(":8080", router))

	cmd.Wait()
}
