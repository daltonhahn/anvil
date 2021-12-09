package anvil

import (
	"crypto/tls"
	"sync"
	"log"
	"net/http"
	"io/ioutil"
	"crypto/x509"
	"time"
	"os"

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
		log.Printf("certman: can't load cert or key file: %v\n", err)
	}
	cw.watching = make(chan bool)
	go cw.run()
	return nil
}

func (cw *CertWatcher) load() error {
	security.ReadSecConfig()
	var err error
        caCertPool := x509.NewCertPool()
	for _, fp := range security.SecConf.CACert {
		caCert, err := ioutil.ReadFile(fp)
		if err != nil {
			log.Printf("Read file error #%v", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}
	tlsConfig = &tls.Config{}
        tlsConfig.Certificates = make([]tls.Certificate, 1)

	tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(security.SecConf.TLSCert, security.SecConf.TLSKey)
        if err != nil {
                log.Fatal(err)
        }
        tlsConfig.BuildNameToCertificate()

	cw.mu.Lock()
	cw.keyPairs = tlsConfig.Certificates
	cw.caCerts = caCertPool
	cw.mu.Unlock()
	if initFlag == 0 {
		initFlag = 1
	} else {
		sigHandle <- struct{}{}
	}
	return err
}

func (cw *CertWatcher) run() {
	deadline := time.Now().Add(10*time.Second)
	loop:
	for {
		select {
		case <-cw.watching:
			break loop
		case <-cw.watcher.Events:
			if time.Now().After(deadline) {
				deadline = time.Now()
				if err := cw.load(); err != nil {
					log.Printf("certman: can't load cert or key file: %v\n", err)
				}
			}
		case err := <-cw.watcher.Errors:
			log.Printf("certman: error watching files: %v\n", err)
		}
	}
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
        dump, err := os.OpenFile("/dev/null", os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
                log.Println("Failed to open dev null")
                log.Println(err)
        }
        nullLog := log.New(dump, "", log.LstdFlags)
	server = &http.Server{
		MaxHeaderBytes: 1 << 20,
		Addr: ":443",
		TLSConfig: cw.GetConfig(),
		Handler: anv_router,
		ErrorLog: nullLog,
	}
	rotFlag = true
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Println(err)
	}
	return nil
}
