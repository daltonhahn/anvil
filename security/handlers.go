package security

import (
	"net/http"
	"io"
	"io/ioutil"
	"log"
	"fmt"
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
	CACert	string		`yaml:"cacert,omitempty"`
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
	ReadSecConfig()
	res1, err1 := TLSGetReqSvc(target, path, origin, 0)
	if err1 != nil {
		//fmt.Println("GET failed on config 1")
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
	ReadSecConfig()

	if path == "/anvil/rotation" {
		fmt.Println("Making POST req")
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println("Body read failure")
	}
	if path == "/anvil/rotation" {
		fmt.Println("Read body bytes and making request with config 1")
	}
	res1, err1 := TLSPostReqSvc(target, path, origin, options, string(b), 0)
	if err1 != nil {
		if path == "/anvil/rotation" {
			fmt.Println("Got an error from the first config request")
		}
		//fmt.Println("POST failed on config 1")
		if len(SecConf) < 2 {
			return &http.Response{},err1
		} else {
			if path == "/anvil/rotation" {
				fmt.Println("Read body bytes and making request with config 1")
			}
			res2, err2 := TLSPostReqSvc(target, path, origin, options, string(b), 1)
			if err2 != nil {
				if path == "/anvil/rotation" {
					fmt.Println("Got an error from the second config request")
				}
				return &http.Response{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}
