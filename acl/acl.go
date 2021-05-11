package acl

import (
	/*
        "crypto/aes"
        "crypto/cipher"
        "crypto/rand"
        //"crypto/tls"
        //"crypto/x509"
        //"errors"
        "fmt"
        "io"
        "io/ioutil"
        "log"
        "gopkg.in/yaml.v2"
	*/
	"github.com/daltonhahn/anvil/service"
)

type ACLEntry struct {
	TokenName	string	`yaml:"name,omitempty"`
	SecretValue	string
	CreationTime	string
	ExpirationTime	string
	ServiceList	[]service.Service
}
