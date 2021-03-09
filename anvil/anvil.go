package anvil

import (
	"fmt"
	"time"
	"net/http"
	"github.com/julienschmidt/httprouter"

)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func GetCatalog(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Catalog at " + dt.String() + "\n"))
}

func GetNodeServices(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node_name := params.ByName("node")
	dt := time.Now()
	fmt.Fprint(w, ("Retrieving Services for Node: " + node_name + "\nCurrent Time: " + dt.String() + "\n"))
}
