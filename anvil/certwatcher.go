package anvil

import (
	"crypto/tls"
	"sync"
	//"os"
	"log"
	"net/http"
	"fmt"
	"io/ioutil"
	"crypto/x509"
	//"context"
	//"time"

	"github.com/gorilla/mux"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/daltonhahn/anvil/security"
)

var tlsConfig *tls.Config

var initFlag = 0

var server *http.Server

var sigHandle = make(chan struct{}, 1)

type CertWatcher struct {
	mu       sync.RWMutex
	conf	 *tls.Config
	keyPairs	[]tls.Certificate
	caCerts		*x509.CertPool
	watcher  *fsnotify.Watcher
	watching chan bool
}

func New() (*CertWatcher, error) {
	cw := &CertWatcher{
		mu:       sync.RWMutex{},
	}
	return cw, nil
}

func (cw *CertWatcher) Watch() error {
	var err error
	if cw.watcher, err = fsnotify.NewWatcher(); err != nil {
		return errors.Wrap(err, "certman: can't create watcher")
	}
	if err = cw.watcher.Add("/root/anvil/config/test_config.yaml"); err != nil {
		return errors.Wrap(err, "certman: can't watch cert file")
	}
	if err := cw.load(); err != nil {
		fmt.Printf("certman: can't load cert or key file: %v\n", err)
	}
	fmt.Printf("watching for config change\n")
	cw.watching = make(chan bool)
	go cw.run()
	return nil
}

func (cw *CertWatcher) load() error {
	fmt.Println("Landed in load()")

	security.ReadSecConfig()
	caCert, err := ioutil.ReadFile(security.SecConf[0].CACert)
        if err != nil {
                fmt.Println("Unable to read config 1 ca.crt")
                log.Printf("Read file error #%v", err)
        }
        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig = &tls.Config{}

        if len(security.SecConf) >= 2 {
                tlsConfig.Certificates = make([]tls.Certificate, 2)
        } else {
                tlsConfig.Certificates = make([]tls.Certificate, 1)
        }

        tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(security.SecConf[0].TLSCert, security.SecConf[0].TLSKey)
        if err != nil {
                log.Fatal(err)
        }
        if len(security.SecConf) >= 2 {
                caCert, err := ioutil.ReadFile(security.SecConf[1].CACert)
                if err != nil {
                        fmt.Println("Unable to read config 2 ca.crt")
                        log.Printf("Read file error #%v", err)
                }
                caCertPool.AppendCertsFromPEM(caCert)
                tlsConfig.Certificates[1], err = tls.LoadX509KeyPair(security.SecConf[1].TLSCert, security.SecConf[1].TLSKey)
                if err != nil {
                        log.Fatal(err)
                }
	}
        tlsConfig.BuildNameToCertificate()
	fmt.Println("Loaded certs")

	cw.mu.Lock()
	cw.keyPairs = tlsConfig.Certificates
	cw.caCerts = caCertPool
	cw.mu.Unlock()
	fmt.Printf("certman: certificate and key loaded\n")
	if initFlag == 0 {
		fmt.Println("first initialization, resetting flag")
		initFlag = 1
	} else {
		fmt.Println("other time, sending shutdown signal")
		sigHandle <- struct{}{}
	}
	return err
}

func (cw *CertWatcher) run() {
loop:
	for {
		select {
		case <-cw.watching:
			break loop
		case event := <-cw.watcher.Events:
			fmt.Printf("certman: watch event: %v\n", event)
			if err := cw.load(); err != nil {
				fmt.Printf("certman: can't load cert or key file: %v\n", err)
			}
		case err := <-cw.watcher.Errors:
			fmt.Printf("certman: error watching files: %v\n", err)
		}
	}
	fmt.Printf("certman: stopped watching\n")
	cw.watcher.Close()
}

func (cw *CertWatcher) GetCertificate(hello *tls.ClientHelloInfo) ([]tls.Certificate, error) {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.keyPairs, nil
}

func (cw*CertWatcher) Stop() {
	cw.watching <- false
}

func (cw *CertWatcher) GetConfig() (*tls.Config) {
        cw.conf = &tls.Config{
                ClientAuth:             tls.RequireAndVerifyClientCert,
                ClientCAs:              cw.caCerts,
		Certificates:		cw.keyPairs,
        }
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.conf
}

func (cw *CertWatcher) startNewServer(anv_router *mux.Router) error {
	server = &http.Server{
		MaxHeaderBytes: 1 << 20,
		Addr: ":443",
		TLSConfig: cw.GetConfig(),
		Handler: anv_router,
	}
	fmt.Println("Starting up new server")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		fmt.Println(err)
	}
	return nil
}
