package anvil

import (
	"net/http"
	"encoding/json"
	"bytes"
	"log"
	"io/ioutil"
	"os"
	//"fmt"

	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/router"
	"github.com/daltonhahn/anvil/acl"
	//"github.com/daltonhahn/anvil/security"
)

func CheckStatus() bool {
	hname, err := os.Hostname()
        if err != nil {
                log.Fatalln("Unable to get hostname")
        }
	//_, err = security.TLSGetReq(hname, "/anvil")
	_, err = http.Get("http://" + hname + ":443/anvil")
	if err == nil {
		return true
	} else {
		return false
	}
}

func Submit(filepath string) {
	acl.ACLIngest(filepath)
	/*
	hname, err := os.Hostname()
        if err != nil {
                log.Fatalln("Unable to get hostname")
        }
	var command struct {
		Command	string `json:"command"`
	}
	command.Command = comm
	postBody, _ := json.Marshal(command)
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("http://"+hname+":443/anvil/raft/pushACL", "application/json", responseBody)
        if err != nil {
                log.Fatalln("Unable to post content")
        }
        defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                log.Fatalln("Unable to read received content")
        }
	fmt.Println(string(body))
	*/
}

func Join(target string) {
	//Collect all of the current info you have from your local catalog
	hname, err := os.Hostname()
        if err != nil {
                log.Fatalln("Unable to get hostname")
        }
	//resp, err := security.TLSGetReq(hname, "/anvil/catalog")
	resp, err := http.Get("http://" + hname + ":443/anvil/catalog")
	if err != nil {
		log.Fatalln("Unable to get response")
	}

	body, err := ioutil.ReadAll(resp.Body)
	var receivedStuff router.Message

	err = json.Unmarshal(body, &receivedStuff)
	if err != nil {
		log.Fatalln("Unable to decode JSON")
	}

	postBody, _ := json.Marshal(receivedStuff)

	responseBody := bytes.NewBuffer(postBody)
	//resp, err = security.TLSPostReq(target, "/anvil/catalog/register", "application/json", responseBody)
	resp, err = http.Post("http://" + target + ":443/anvil/catalog/register", "application/json", responseBody)
	if err != nil {
		log.Fatalln("Unable to post content")
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Unable to read received content")
	}

	var respMsg router.Message
	err = json.Unmarshal(body, &respMsg)
	if err != nil {
		log.Fatalln("Unable to process response JSON")
	}

	var tempCatalog catalog.Catalog
	for _, ele := range respMsg.Nodes {
		tempCatalog.AddNode(ele)
		for _, svc := range respMsg.Services {
			if (ele.Address == svc.Address) {
				tempCatalog.AddService(svc)
			}
		}
		var localPost router.Message
		localPost.NodeName = ele.Name
		localPost.Services = tempCatalog.Services
		localPost.NodeType = ele.Type
		postBody, _ = json.Marshal(localPost)
		responseBody = bytes.NewBuffer(postBody)
		// Marshal the struct into a postable message
		hname, err := os.Hostname()
		if err != nil {
			log.Fatalln("Unable to get hostname")
		}
		http.Post("http://"+hname+":443/anvil/catalog/register", "application/json", responseBody)
		tempCatalog = catalog.Catalog{}
	}
}
