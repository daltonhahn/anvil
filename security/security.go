package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"
	"net/http"
)

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
    text := []byte(plaintext)
    key := []byte(SecConf.Key)

    // generate a new aes cipher using our 32 byte long key
    c, err := aes.NewCipher(key)
    // if there are any errors, handle them
    if err != nil {
        fmt.Println(err)
    }

    // gcm or Galois/Counter Mode, is a mode of operation
    // for symmetric key cryptographic block ciphers
    // - https://en.wikipedia.org/wiki/Galois/Counter_Mode
    gcm, err := cipher.NewGCM(c)
    // if any error generating new GCM
    // handle them
    if err != nil {
        fmt.Println(err)
    }

    // creates a new byte array the size of the nonce
    // which must be passed to Seal
    nonce := make([]byte, gcm.NonceSize())
    // populates our nonce with a cryptographically secure
    // random sequence
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        fmt.Println(err)
    }

    // here we encrypt our text using the Seal function
    // Seal encrypts and authenticates plaintext, authenticates the
    // additional data and appends the result to dst, returning the updated
    // slice. The nonce must be NonceSize() bytes long and unique for all
    // time, for a given key.
    return []byte(gcm.Seal(nonce, nonce, text, nil))
}

func DecData(input_ciphertext string) []byte {
    key := []byte(SecConf.Key)
    ciphertext := []byte(input_ciphertext)
    c, err := aes.NewCipher(key)
    if err != nil {
        fmt.Println(err)
    }

    gcm, err := cipher.NewGCM(c)
    if err != nil {
        fmt.Println(err)
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        fmt.Println(err)
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        fmt.Println(err)
    }
    return plaintext
}

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
