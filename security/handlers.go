package security

import (
	"net/http"
	"io"
	"os"
	"time"

	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"

	"github.com/daltonhahn/anvil/service"
)

type ACLEntry struct {
	TokenName	string	`yaml:"name,omitempty"`
	SecretValue	string
	CreationTime	string
	ExpirationTime	string
	ServiceList	[]service.Service
}

var SecConf = []SecConfig{}

type TokMap struct {
	ServiceName	string	`yaml:"sname,omitempty"`
	TokenVal	string	`yaml:"tval,omitempty"`
}

type SecConfig struct {
	Key	string		`yaml:"key,omitempty"`
	CACert	[]string	`yaml:"cacert,omitempty"`
	TLSCert	string		`yaml:"tlscert,omitempty"`
	TLSKey	string		`yaml:"tlskey,omitempty"`
	Tokens	[]TokMap	`yaml:"tokens,omitempty"`
}


func ReadSecConfig() {
	yamlFile, err := ioutil.ReadFile("/root/anvil/config/test_config.yaml")
        if err != nil {
		//fmt.Println("Unable to read test_config")
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
}

func EncData(plaintext string) ([]byte, error) {
	ReadSecConfig()
	res1, err1 := EncDataSvc(plaintext, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return []byte{}, err1
		} else {
			res2, err2 := EncDataSvc(plaintext, 1)
			if err2 != nil {
				return []byte{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func DecData(input_ciphertext string) ([]byte,error) {
	ReadSecConfig()
	res1, err1 := DecDataSvc(input_ciphertext, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return []byte{}, err1
		} else {
			res2, err2 := DecDataSvc(input_ciphertext, 1)
			if err2 != nil {
				return []byte{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func TLSGetReq(target string, path string, origin string) (*http.Response,error) {
	if HTTPStat() == false {
		time.Sleep(1*time.Second)
	}
	ReadSecConfig()
	res1, err1 := TLSGetReqSvc(target, path, origin, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return &http.Response{},err1
		} else {
			res2, err2 := TLSGetReqSvc(target, path, origin, 1)
			if err2 != nil {
				return &http.Response{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func TLSPostReq(target string, path string, origin string, options string, body io.Reader) (*http.Response, error) {
	if HTTPStat() == false {
		time.Sleep(1*time.Second)
	}
	ReadSecConfig()
	b, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println("Body read failure")
	}
	res1, err1 := TLSPostReqSvc(target, path, origin, options, string(b), 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return &http.Response{},err1
		} else {
			res2, err2 := TLSPostReqSvc(target, path, origin, options, string(b), 1)
			if err2 != nil {
				return &http.Response{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func HTTPStat() bool {
        hname, _ := os.Hostname()
        resp, err := TLSGetReqSvc(hname, "/anvil/", "", 0)
        if err != nil || resp.StatusCode != http.StatusOK {
		resp, err := TLSGetReqSvc(hname, "/anvil/", "", 1)
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
        }
        return true
}
