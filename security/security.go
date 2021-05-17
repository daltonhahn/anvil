package security

import (
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
	//"net/http"

	"github.com/daltonhahn/anvil/service"
)

type ACLEntry struct {
	TokenName	string	`yaml:"name,omitempty"`
	SecretValue	string
	CreationTime	string
	ExpirationTime	string
	ServiceList	[]service.Service
}

var SecConf = new(SecConfig)

type SecConfig struct {
	Key	string `yaml:"key,omitempty"`
	CACert	string `yaml:"cacert,omitempty"`
	TLSCert	string `yaml:"tlscert,omitempty"`
	TLSKey	string `yaml:"tlskey,omitempty"`
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

func EncData(plaintext string) []byte {
	fmt.Println("In EncData")
    ReadSecConfig()
    text := []byte(plaintext)
    key := []byte(SecConf.Key)

    c, err := aes.NewCipher(key)
    if err != nil {
        fmt.Println(err)
    }
    gcm, err := cipher.NewGCM(c)
    if err != nil {
        fmt.Println(err)
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        fmt.Println(err)
    }
    //fmt.Println("\tENC NONCE: %x", nonce)
    //fmt.Println("ENC MSG: %x", gcm.Seal(nonce, nonce, text, nil))
    return []byte(gcm.Seal(nonce, nonce, text, nil))
}

func DecData(input_ciphertext string) []byte {
	fmt.Println("In DecData")
    ReadSecConfig()
    key := []byte(SecConf.Key)
    data := []byte(input_ciphertext)
    c, err := aes.NewCipher(key)
    if err != nil {
        fmt.Println(err)
    }

    gcm, err := cipher.NewGCM(c)
    if err != nil {
        fmt.Println(err)
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        fmt.Println(err)
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    //fmt.Println("\tDEC NONCE: %x", nonce)
    //fmt.Println("DEC CIPHERTXT: %x", data)
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        fmt.Println(err)
    }
    return plaintext
}

/*
func TLSGetReq(target string, path string) (*http.Response,error) {
	ReadSecConfig()
	caCertPath := SecConf.CACert
	caCert, err := ioutil.ReadFile(caCertPath)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert,err := tls.LoadX509KeyPair(SecConf.TLSCert, SecConf.TLSKey)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	resp, err := client.Get("https://"+target+path)
	if err != nil {
		log.Println(err)
		return &http.Response{}, errors.New("No HTTPS response")
	}
	return resp, nil
}

func TLSPostReq(target string, path string, options string, body io.Reader) (*http.Response, error) {
	ReadSecConfig()
        caCertPath := SecConf.CACert
        caCert, err := ioutil.ReadFile(caCertPath)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
        cert,err := tls.LoadX509KeyPair(SecConf.TLSCert, SecConf.TLSKey)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }

        client := &http.Client{
                Transport: &http.Transport{
                        TLSClientConfig: &tls.Config{
                                RootCAs:      caCertPool,
                                Certificates: []tls.Certificate{cert},
                        },
                },
        }

        resp, err := client.Post("https://"+target+path, options, body)
        if err != nil {
                log.Println(err)
                return &http.Response{}, errors.New("No HTTPS response")
        }
        return resp, nil
}
*/
