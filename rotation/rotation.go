package rotation

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"path/filepath"
	"log"
	"bytes"
	"encoding/json"
	"net/http"
        "gopkg.in/yaml.v2"
        "strconv"
        "reflect"

	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/catalog"
)

type SecConfig struct {
        Key     string          `yaml:"key,omitempty"`
        CACert  string          `yaml:"cacert,omitempty"`
        TLSCert string          `yaml:"tlscert,omitempty"`
        TLSKey  string          `yaml:"tlskey,omitempty"`
        Tokens  []TokMap        `yaml:"tokens,omitempty"`
}

type TokMap struct {
        ServiceName     string  `yaml:"sname,omitempty"`
        TokenVal        string  `yaml:"tval,omitempty"`
}

var SecConf []SecConfig

type FPMess struct {
        FilePath        string
}

func CollectFiles(iter string, nodeName string) bool {
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
	resp, err := security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

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

	fMess = &FPMess{FilePath: nodeName+"/acl.yaml"}
	jsonData, err = json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal = bytes.NewBuffer(jsonData)
	resp, err = security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

	out, err = os.Create("/root/anvil/config/acls/"+iter+"/acl.yaml")
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

	fMess = &FPMess{FilePath: nodeName+"/"+nodeName+".key"}
	jsonData, err = json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal = bytes.NewBuffer(jsonData)
	resp, err = security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

	out, err = os.Create("/root/anvil/config/certs/"+iter+"/"+nodeName+".key")
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

	fMess = &FPMess{FilePath: nodeName+"/"+nodeName+".crt"}
	jsonData, err = json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal = bytes.NewBuffer(jsonData)
	resp, err = security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

	out, err = os.Create("/root/anvil/config/certs/"+iter+"/"+nodeName+".crt")
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

	fMess = &FPMess{FilePath: nodeName+"/ca.crt"}
	jsonData, err = json.Marshal(fMess)
	if err != nil {
		log.Fatalln("Unable to marshal JSON")
	}
	postVal = bytes.NewBuffer(jsonData)
	resp, err = security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

	out, err = os.Create("/root/anvil/config/certs/"+iter+"/ca.crt")
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

	AdjustConfig()
	return true
}

func AdjustConfig() {
        iterMap := getDirMap()
        //fmt.Printf("%v\n", iterMap)

	firstFlag := 0
        cmpList := []int{}
        for _, list := range iterMap {
		if firstFlag == 0 {
			cmpList = list
			firstFlag = 1
		} else {
                        if !reflect.DeepEqual(cmpList, list) {
                                fmt.Println("We've got problems")
                        }
                }
                //fmt.Printf("%v\n", list)
        }

        if len(cmpList) < 2 {
                rewriteYaml(cmpList[0], 0)
        } else {
                rewriteYaml(cmpList[len(cmpList)-1], cmpList[len(cmpList)-2])
        }
}

func updateRunningConfig(yamlOut string) {
	f, err := os.OpenFile("/root/anvil/config/test_config.yaml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	yamlOut = "---\n" + yamlOut
	_,err = f.Write([]byte(yamlOut))
        if err != nil {
                fmt.Println(err)
        }
	defer f.Close()
	// Adjusting the config file itself will solve all OUTBOUND PROBLEMS and
	// a lot of the INBOUND decryption problems, but will not solve the
	// ACTIVE TLS instance problem

	// THIS FUNCTION SHOULD INTERRUPT AND ADJUST OR MODIFY THE EXISTING
	// TLS CONFIG SO THAT WE CAN HANDLE NEW INCOMING REQUESTS

}

func rewriteYaml(indA int, indB int) {
        yamlFile, err := ioutil.ReadFile("/root/anvil/config/test_config.yaml")
        if err != nil {
		fmt.Println("Unable to read test_config")
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
        //fmt.Printf("%v\n", SecConf)
        //fmt.Printf("------\n")

        hname, _ := os.Hostname()
        listSecConf := []SecConfig{}
	var yamlOut []byte

        if indB == 0 && indA == 0 {
		b, err := ioutil.ReadFile("/root/anvil/config/gossip/0/gossip.key")
		if err != nil {
			panic(err)
		}
		gKey := string(b)
                tokMap := readACLFile("/root/anvil/config/acls/0/test.yaml")
                tmpSecConf := SecConfig {
                        Key: gKey,
                        CACert: "/root/anvil/config/certs/0/ca.crt",
                        TLSCert: "/root/anvil/config/certs/0/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/0/"+hname+".key",
                        Tokens: tokMap,
                }
                listSecConf = []SecConfig{tmpSecConf}

                yamlOut, err = yaml.Marshal(listSecConf)
                if err != nil {
                        panic(err)
                }
                //fmt.Printf("%v\n", string(yamlOut))
        } else {
                strA := strconv.Itoa(indA)
                strB := strconv.Itoa(indB)

                tMapA := readACLFile("/root/anvil/config/acls/"+strA+"/acl.yaml")
                tMapB := readACLFile("/root/anvil/config/acls/"+strB+"/acl.yaml")
		b, err := ioutil.ReadFile("/root/anvil/config/gossip/"+strA+"/gossip.key")
		if err != nil {
			panic(err)
		}
		gKeyA := string(b)
                sConfA := SecConfig {
                        Key: gKeyA,
                        CACert: "/root/anvil/config/certs/"+strA+"/ca.crt",
                        TLSCert: "/root/anvil/config/certs/"+strA+"/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/"+strA+"/"+hname+".key",
                        Tokens: tMapA,
                }
		b, err = ioutil.ReadFile("/root/anvil/config/gossip/"+strB+"/gossip.key")
		if err != nil {
			panic(err)
		}
		gKeyB := string(b)
                sConfB := SecConfig {
                        Key: gKeyB,
                        CACert: "/root/anvil/config/certs/"+strB+"/ca.crt",
                        TLSCert: "/root/anvil/config/certs/"+strB+"/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/"+strB+"/"+hname+".key",
                        Tokens: tMapB,
                }
                listSecConf = []SecConfig{sConfA, sConfB}
                yamlOut, err = yaml.Marshal(listSecConf)
                if err != nil {
                        panic(err)
                }
                //fmt.Printf("%v\n", string(yamlOut))
        }
	updateRunningConfig(string(yamlOut))
}

func getDirMap() map[string][]int {
        aclIters, err := ioutil.ReadDir("/root/anvil/config/acls")
        if err != nil {
                log.Println(err)
        }
        gossipIters, err := ioutil.ReadDir("/root/anvil/config/gossip")
        if err != nil {
                log.Println(err)
        }
        certIters, err := ioutil.ReadDir("/root/anvil/config/certs")
        if err != nil {
                log.Println(err)
        }

        iterMap := make(map[string][]int)
        for _, f := range aclIters {
                if f.IsDir() {
                        val, err := strconv.Atoi(f.Name())
                        if err != nil {
                                fmt.Println("Unable to convert")
                        }
                        iterMap["acls"] = append(iterMap["acls"], val)
                }
        }
        for _, f := range gossipIters {
                if f.IsDir() {
                        val, err := strconv.Atoi(f.Name())
                        if err != nil {
                                fmt.Println("Unable to convert")
                        }
                        iterMap["gossip"] = append(iterMap["gossip"], val)
                }
        }
        for _, f := range certIters {
                if f.IsDir() {
                        val, err := strconv.Atoi(f.Name())
                        if err != nil {
                                fmt.Println("Unable to convert")
                        }
                        iterMap["certs"] = append(iterMap["certs"], val)
                }
        }

        return iterMap
}

func readACLFile(fpath string) []TokMap {
        retToks := []TokMap{}
        yamlFile, err := ioutil.ReadFile(fpath)
        if err != nil {
		fmt.Println("Unable to read aclfile")
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &retToks)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
        return retToks
}
