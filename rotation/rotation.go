package rotation

import (
	"fmt"
	"os"
	"io"
	"path/filepath"
	"log"
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/catalog"
)

type FPMess struct {
        FilePath        string
}

func CollectFiles(iter string, nodeName string) bool {
	fmt.Printf("THIS IS ITER: %v and I'm %v\n", iter, nodeName)
	newpath := filepath.Join("/root/anvil/", "config/gossip", iter)
        os.MkdirAll(newpath, os.ModePerm)
        newpath = filepath.Join("/root/anvil/", "config/acls", iter)
        os.MkdirAll(newpath, os.ModePerm)
        newpath = filepath.Join("/root/anvil/", "config/certs", iter)
        os.MkdirAll(newpath, os.ModePerm)

	anv_catalog := catalog.GetCatalog()
        qMem := anv_catalog.GetQuorumMem()

	fMess := &FPMess{FilePath: "gossip.key"}
	jsonData, err := json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal := bytes.NewBuffer(jsonData)
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle"+iter, "rotation", "application/json", postVal)

	out, err := os.Create("/root/anvil/config/gossip/"+iter+"/"+fMess.FilePath)
	if err != nil  {
		fmt.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}

	fMess := &FPMess{FilePath: nodeName+"/acl.yaml"}
	jsonData, err := json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal := bytes.NewBuffer(jsonData)
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle"+iter, "rotation", "application/json", postVal)

	out, err := os.Create("/root/anvil/config/acls/"+iter+"/acl.yaml")
	if err != nil  {
		fmt.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}

	fMess := &FPMess{FilePath: nodeName+"/"+nodeName+".key"}
	jsonData, err := json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal := bytes.NewBuffer(jsonData)
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle"+iter, "rotation", "application/json", postVal)

	out, err := os.Create("/root/anvil/config/certs/"+iter+"/"+nodeName+".key")
	if err != nil  {
		fmt.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}

	fMess := &FPMess{FilePath: nodeName+"/"+nodeName+".crt"}
	jsonData, err := json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal := bytes.NewBuffer(jsonData)
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle"+iter, "rotation", "application/json", postVal)

	out, err := os.Create("/root/anvil/config/certs/"+iter+"/"+nodeName+".crt")
	if err != nil  {
		fmt.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}


	/*
	fMess := &FPMess{FilePath: nodeName+"/ca.crt"}
	jsonData, err := json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal := bytes.NewBuffer(jsonData)
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle"+iter, "rotation", "application/json", postVal)

	out, err := os.Create("/root/anvil/config/certs/"+iter+"/ca.crt")
	if err != nil  {
		fmt.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}
	*/

	// Every time this is called, select a random quorum member
	// For loop for 4 files
		// gossip.key
		// ca.crt
		// nodeName.crt
		// nodeName.key
	return true
}



/*
out, err := os.Create("/root/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+f)
                                        if err != nil  {
                                                fmt.Printf("FAILURE OPENING FILE\n")
                                        }
                                        defer out.Close()
                                        defer resp.Body.Close()
                                        if resp.StatusCode != http.StatusOK {
                                                fmt.Errorf("bad status: %s", resp.Status)
                                        }
                                        _, err = io.Copy(out, resp.Body)
                                        if err != nil  {
                                                fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
                                        }
					*/
