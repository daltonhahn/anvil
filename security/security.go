package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"fmt"
)

func EncDataSvc(plaintext string) ([]byte,error) {
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

func DecDataSvc(input_ciphertext string) ([]byte,error) {
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

func TLSGetReqSvc(target string, path string, origin string, prevChain string) (*http.Response,error) {
	caCertPaths := SecConf.CACert
	caCertPool := x509.NewCertPool()
        for _, fp := range caCertPaths {
                caCert, err := ioutil.ReadFile(fp)
                if err != nil {
			log.Println("Read file error in GET REQ SVC")
                }
                caCertPool.AppendCertsFromPEM(caCert)
        }
	cert,err := tls.LoadX509KeyPair(SecConf.TLSCert, SecConf.TLSKey)
        if err != nil {
                return &http.Response{}, errors.New("Unable to read Cert+Key")
        }

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	req, err := http.NewRequest("GET", ("https://"+target+path), nil)
	if (len(strings.Split(path, "/")) >= 3) {
		if (strings.Split(path, "/")[1] != "anvil") {
			fmt.Printf("Complex request path, searching for token\n")
			targetSvc := strings.Split(path, "/")[2]
			bearer := attachToken(origin, targetSvc, prevChain)
			if (len(bearer) >= 1) {
				fmt.Printf("BEARER BEING ATTACHED: %v\n", bearer)
			}
			req.Header.Add("Authorization", bearer)
			fmt.Printf("TARGET: %v -- PATH: %v\n", target, path)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, errors.New("No HTTPS response")
	}

	return resp, nil
}

func TLSPostReqSvc(target string, path string, origin string, options string, body string, prevChain string) (*http.Response, error) {
        caCertPaths := SecConf.CACert
	caCertPool := x509.NewCertPool()
        for _, fp := range caCertPaths {
                caCert, err := ioutil.ReadFile(fp)
                if err != nil {
			log.Println("Read file error in POST REQ SVC")
                }
                caCertPool.AppendCertsFromPEM(caCert)
        }
        cert,err := tls.LoadX509KeyPair(SecConf.TLSCert, SecConf.TLSKey)
        if err != nil {
                return &http.Response{}, errors.New("Unable to read Cert+Key")
        }

        client := &http.Client{
                Transport: &http.Transport{
                        TLSClientConfig: &tls.Config{
                                RootCAs:      caCertPool,
                                Certificates: []tls.Certificate{cert},
                        },
                },
        }

	req, err := http.NewRequest("POST", ("https://"+target+path), strings.NewReader(body))
	req.Header.Set("Content-type", options)
	if (len(strings.Split(path, "/")) >= 3) {
		if (strings.Split(path, "/")[1] != "anvil") {
			fmt.Printf("Complex request path, searching for token\n")
			targetSvc := strings.Split(path, "/")[2]
			bearer := attachToken(origin, targetSvc, prevChain)
			if (len(bearer) >= 1) {
				fmt.Printf("BEARER BEING ATTACHED: %v\n", bearer)
			}
			req.Header.Add("Authorization", bearer)
		}
	}

        resp, err := client.Do(req)
        if err != nil {
                return &http.Response{}, errors.New("No HTTPS response")
        }
	client.CloseIdleConnections()

        return resp, nil
}

func attachToken(originSvc string, targetSvc string, prevChain string) string {
	// Take prevChain data and parse it out
	// Format everything into a JSON string
	if (len(prevChain) <= 0) {
		for _, ele := range SecConf.Tokens {
			if ele.ServiceName == targetSvc {
				return "[{\"token\":\""+ele.TokenVal+"\",\"service\":\""+targetSvc+"\"}]"
			}
		}
	} else {
		for _, ele := range SecConf.Tokens {
			if ele.ServiceName == targetSvc {
				fmt.Printf("My target: %v -- Checking target: %v\n", targetSvc, ele.ServiceName)
				fmt.Printf("Matched a service with the target\n")
				return prevChain[:len(prevChain)-3] + ",{\"token\":"+ele.TokenVal+",\"service\":"+targetSvc+"}]}"
			}
		}
	}
	return ""
}
