package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	//"fmt"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"
	"net/http"

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
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
}

func EncData(plaintext string) ([]byte,error) {
    ReadSecConfig()
    text := []byte(plaintext)
    key := []byte(SecConf.Key)

    c, err := aes.NewCipher(key)
    if err != nil {
	return []byte{}, err
    }
    gcm, err := cipher.NewGCM(c)
    if err != nil {
	return []byte{}, err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
	return []byte{}, err
    }
    return []byte(gcm.Seal(nonce, nonce, text, nil)),nil
}

func DecData(input_ciphertext string) ([]byte,error) {
    ReadSecConfig()
    key := []byte(SecConf.Key)
    data := []byte(input_ciphertext)
    c, err := aes.NewCipher(key)
    if err != nil {
	return []byte{}, err
    }

    gcm, err := cipher.NewGCM(c)
    if err != nil {
	return []byte{}, err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
	return []byte{}, err
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
	return []byte{}, err
    }
    return plaintext,nil
}

func TLSGetReq(target string, path string, origin string) (*http.Response,error) {
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

	bearer := attachToken(origin)
	req, err := http.NewRequest("GET", ("https://"+target+path), nil)
	req.Header.Add("Authorization", bearer)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return &http.Response{}, errors.New("No HTTPS response")
	}
	return resp, nil
}

func TLSPostReq(target string, path string, origin string, options string, body io.Reader) (*http.Response, error) {
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

	bearer := attachToken(origin)
	req, err := http.NewRequest("POST", ("https://"+target+path), body)
	req.Header.Set("Content-type", options)
	req.Header.Add("Authorization", bearer)

        resp, err := client.Do(req)
        if err != nil {
                log.Println(err)
                return &http.Response{}, errors.New("No HTTPS response")
        }
        return resp, nil
}

func attachToken(originSvc string) string {
	for _, ele := range SecConf.Tokens {
		if ele.ServiceName == originSvc {
			return ele.TokenVal
		}
	}
	return ""
}
