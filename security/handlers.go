package security

import (
	"net/http"
	"io"

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

var SecConf = SecConfig{}

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
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
}

func EncData(plaintext string) ([]byte, error) {
	ReadSecConfig()
	res1, err1 := EncDataSvc(plaintext)
	if err1 != nil {
		return []byte{}, err1
	}
	return res1, nil
}

func DecData(input_ciphertext string) ([]byte,error) {
	ReadSecConfig()
	res1, err1 := DecDataSvc(input_ciphertext)
	if err1 != nil {
		return []byte{}, err1
	}
	return res1, nil
}

func TLSGetReq(target string, path string, origin string, prevChain string) (*http.Response,error) {
	ReadSecConfig()
	res1, err1 := TLSGetReqSvc(target, path, origin, prevChain)
	if err1 != nil {
		return &http.Response{},err1
	}
	return res1, nil
}

func TLSPostReq(target string, path string, origin string, options string, body io.Reader, prevChain string) (*http.Response, error) {
	ReadSecConfig()
	b, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println("Body read failure")
	}
	res1, err1 := TLSPostReqSvc(target, path, origin, options, string(b), prevChain)
	if err1 != nil {
		return &http.Response{},err1
	}
	return res1, nil
}
