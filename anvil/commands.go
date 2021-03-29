package anvil

import (
	"net/http"
	"encoding/json"
	"bytes"
	"log"
	"io/ioutil"

	"github.com/daltonhahn/anvil/catalog"
	"github.com/daltonhahn/anvil/router"
//	"github.com/daltonhahn/anvil/raft"
)

func CheckStatus() bool {
	resp, err := http.Get("http://localhost/anvil")
	if resp != nil && err == nil {
		return true
	} else {
		return false
	}
}

/*
func SendVoteReq(string target, raft.RequestVoteArgs args) (error, raft.RequestVoteReply) {
	postBody, _ := json.Marshal(args)
	resp, err := http.Post("http://" + target + "/anvil/raft/requestvote", "application/json", postBody)
	if err != nil {
		log.Fatalln("Unable to post content")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		log.Fatalln("Unable to read received content")
        }
        var rv_reply raft.RequestVoteReply
        err = json.Unmarshal(b, &rv_reply)
        if err != nil {
		log.Fatalln("Unable to process response JSON")
        }
	return nil, rv_reply
}
*/

func Join(target string) {
	//Collect all of the current info you have from your local catalog
	resp, err := http.Get("http://localhost/anvil/catalog")
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
	resp, err = http.Post("http://" + target + "/anvil/catalog/register", "application/json", responseBody)
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
		http.Post("http://localhost/anvil/catalog/register", "application/json", responseBody)
		tempCatalog = catalog.Catalog{}
	}
}
