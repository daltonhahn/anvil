package rotation

import (
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
	"strings"
        "reflect"

	"github.com/daltonhahn/anvil/security"
	"github.com/daltonhahn/anvil/catalog"
)

type SecConfig struct {
        Key     string          `yaml:"key,omitempty"`
        CACert  []string          `yaml:"cacert,omitempty"`
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

func CollectFiles(iter string, nodeName string, qMems []string) bool {
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
		log.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		log.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
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
		log.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		log.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
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
		log.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		log.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
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
		log.Printf("FAILURE OPENING FILE\n")
	}
	defer out.Close()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		log.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
	}

	for _, ele := range qMems {
		fMess = &FPMess{FilePath: nodeName+"/"+ele+".crt"}
		jsonData, err = json.Marshal(fMess)
		if err != nil {
			log.Fatalln("Unable to marshal JSON")
		}
		postVal = bytes.NewBuffer(jsonData)
		resp, err = security.TLSPostReq(qMem, "/service/rotation/bundle/"+iter, "rotation", "application/json", postVal)

		out, err = os.Create("/root/anvil/config/certs/"+iter+"/"+ele+".crt")
		if err != nil  {
			log.Printf("FAILURE OPENING FILE\n")
		}
		defer out.Close()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", resp.Status)
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil  {
			log.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
		}
	}
	return true
}

func AdjustConfig() {
        iterMap := getDirMap()

	firstFlag := 0
        cmpList := []int{}
        for _, list := range iterMap {
		if firstFlag == 0 {
			cmpList = list
			firstFlag = 1
		} else {
                        if !reflect.DeepEqual(cmpList, list) {
                                log.Println("We've got problems")
                        }
                }
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
                log.Println(err)
        }
	defer f.Close()
}

func rewriteYaml(indA int, indB int) {
        yamlFile, err := ioutil.ReadFile("/root/anvil/config/test_config.yaml")
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }

        hname, _ := os.Hostname()
        listSecConf := []SecConfig{}
	var yamlOut []byte

        if indB == 0 && indA == 0 {
		b, err := ioutil.ReadFile("/root/anvil/config/gossip/0/gossip.key")
		if err != nil {
			panic(err)
		}
		gKey := string(b)

		caList := processCAs(0)

                tokMap := readACLFile("/root/anvil/config/acls/0/test.yaml")
                tmpSecConf := SecConfig {
                        Key: gKey,
                        CACert: caList,
                        TLSCert: "/root/anvil/config/certs/0/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/0/"+hname+".key",
                        Tokens: tokMap,
                }
                listSecConf = []SecConfig{tmpSecConf}

                yamlOut, err = yaml.Marshal(listSecConf)
                if err != nil {
                        panic(err)
                }
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

		caListA := processCAs(indA)

                sConfA := SecConfig {
                        Key: gKeyA,
                        CACert: caListA,
                        TLSCert: "/root/anvil/config/certs/"+strA+"/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/"+strA+"/"+hname+".key",
                        Tokens: tMapA,
                }
		b, err = ioutil.ReadFile("/root/anvil/config/gossip/"+strB+"/gossip.key")
		if err != nil {
			panic(err)
		}
		gKeyB := string(b)

		caListB := processCAs(indB)

                sConfB := SecConfig {
                        Key: gKeyB,
                        CACert: caListB,
                        TLSCert: "/root/anvil/config/certs/"+strB+"/"+hname+".crt",
                        TLSKey: "/root/anvil/config/certs/"+strB+"/"+hname+".key",
                        Tokens: tMapB,
                }
                listSecConf = []SecConfig{sConfA, sConfB}
                yamlOut, err = yaml.Marshal(listSecConf)
                if err != nil {
                        panic(err)
                }
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
                                log.Println("Unable to convert")
                        }
                        iterMap["acls"] = append(iterMap["acls"], val)
                }
        }
        for _, f := range gossipIters {
                if f.IsDir() {
                        val, err := strconv.Atoi(f.Name())
                        if err != nil {
                                log.Println("Unable to convert")
                        }
                        iterMap["gossip"] = append(iterMap["gossip"], val)
                }
        }
        for _, f := range certIters {
                if f.IsDir() {
                        val, err := strconv.Atoi(f.Name())
                        if err != nil {
                                log.Println("Unable to convert")
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
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &retToks)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
        return retToks
}

func processCAs(iter int) []string {
	hname, _ := os.Hostname()
	topLvl, err := ioutil.ReadDir("/root/anvil/config/certs/"+strconv.Itoa(iter))
	if err != nil {
		log.Println(err)
	}
	var retList []string
	for _, f := range topLvl {
		if !strings.Contains(f.Name(), ".key") && f.Name() != hname+".crt" {
			retList = append(retList, ("/root/anvil/config/certs/"+strconv.Itoa(iter)+"/"+f.Name()))
		}
	}
	if err != nil {
		log.Fatalln("Unable to get hostname")
	}
	resp, err := security.TLSGetReq(hname, "/anvil/type", "")
	if err != nil {
		log.Fatalln("Unable to retrieve node type")
	}
	body, err := ioutil.ReadAll(resp.Body)
	nodeType := string(body)
	if nodeType == "server" {
		retList = append(retList, "/root/anvil/config/certs/"+strconv.Itoa(iter)+"/"+hname+".crt")
	}
	return retList
}
