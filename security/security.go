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
	"net/http"
)

func EncDataSvc(plaintext string, confNum int) ([]byte,error) {
    text := []byte(plaintext)
    key := []byte(SecConf[confNum].Key)

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

func DecDataSvc(input_ciphertext string, confNum int) ([]byte,error) {
    key := []byte(SecConf[confNum].Key)
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

func TLSGetReqSvc(target string, path string, origin string, confNum int) (*http.Response,error) {
	caCertPath := SecConf[confNum].CACert
	caCert, err := ioutil.ReadFile(caCertPath)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert,err := tls.LoadX509KeyPair(SecConf[confNum].TLSCert, SecConf[confNum].TLSKey)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
                                InsecureSkipVerify:     true,
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
		return &http.Response{}, errors.New("No HTTPS response")
	}

	return resp, nil
}

func TLSPostReqSvc(target string, path string, origin string, options string, body io.Reader, confNum int) (*http.Response, error) {
        caCertPath := SecConf[confNum].CACert
        caCert, err := ioutil.ReadFile(caCertPath)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
        cert,err := tls.LoadX509KeyPair(SecConf[confNum].TLSCert, SecConf[confNum].TLSKey)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }

        client := &http.Client{
                Transport: &http.Transport{
                        TLSClientConfig: &tls.Config{
                                InsecureSkipVerify:     true,
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
                return &http.Response{}, errors.New("No HTTPS response")
        }

        return resp, nil
}

func attachToken(originSvc string) string {
	for _, ele := range SecConf[0].Tokens {
		if ele.ServiceName == originSvc {
			return ele.TokenVal
		}
	}
	return ""
}
