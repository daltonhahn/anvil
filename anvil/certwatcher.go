package anvil

import (
	"crypto/tls"
	"sync"
	"crypto/x509"
	"io/ioutil"
	"log"
	//"fmt"


	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/daltonhahn/anvil/security"
)

var tlsConfig *tls.Config

type CertWatcher struct {
        mu       sync.RWMutex
        conf	 *tls.Config
        keyPairs []tls.Certificate
        watcher  *fsnotify.Watcher
        watching chan bool
}

func New() (*CertWatcher, error) {
        cw := &CertWatcher{
                mu:		sync.RWMutex{},
		conf:		tlsConfig,
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
		panic(err)
        }
        cw.watching = make(chan bool)
        go cw.run()
        return nil
}

func (cw *CertWatcher) load() error {
	//fmt.Println("RELOADING TLS CONFIG")
	security.ReadSecConfig()

        caCertPool := x509.NewCertPool()
        tlsConfig = &tls.Config{
		ClientAuth:             tls.RequireAndVerifyClientCert,
                ClientCAs:              caCertPool,
        }
        if len(security.SecConf) >= 2 {
                tlsConfig.Certificates = make([]tls.Certificate, 2)
                caCert, err := ioutil.ReadFile(security.SecConf[1].CACert)
                if err != nil {
                        log.Printf("Read file error #%v", err)
                }
                caCertPool.AppendCertsFromPEM(caCert)
                tlsConfig.Certificates[1], err = tls.LoadX509KeyPair(security.SecConf[1].TLSCert, security.SecConf[1].TLSKey)
                if err != nil {
                        log.Fatal(err)
                }
        } else {
                tlsConfig.Certificates = make([]tls.Certificate, 1)
        }

	caCert, err := ioutil.ReadFile(security.SecConf[0].CACert)
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        caCertPool.AppendCertsFromPEM(caCert)

        tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(security.SecConf[0].TLSCert, security.SecConf[0].TLSKey)
        if err != nil {
                log.Fatal(err)
        }
        tlsConfig.BuildNameToCertificate()

	/*
	fmt.Printf("%v\n", tlsConfig.Certificates)
	fmt.Printf("----------\n")
	fmt.Printf("%v\n", tlsConfig.Certificates[0])
	fmt.Printf("----------\n")
	fmt.Printf("%v\n", tlsConfig.Certificates[1])
	*/

	cw.mu.Lock()
	cw.conf = tlsConfig
	cw.keyPairs = tlsConfig.Certificates
	cw.conf.Certificates = cw.keyPairs
	cw.mu.Unlock()
	//fmt.Println("SHOULD BE RELOADED NOW")

	return err
}

func (cw *CertWatcher) run() {
loop:
        for {
                select {
                case <-cw.watching:
                        break loop
                case <-cw.watcher.Events:
                        if err := cw.load(); err != nil {
				panic(err)
                        }
                case err := <-cw.watcher.Errors:
			panic(err)
                }
        }
        cw.watcher.Close()
}

func (cw *CertWatcher) GetConfig(hello *tls.ClientHelloInfo) (*tls.Config, error) {
        cw.mu.RLock()
        defer cw.mu.RUnlock()
        return cw.conf, nil
}

/*
func (cw *CertWatcher) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
        cw.mu.RLock()
        defer cw.mu.RUnlock()
        return cw.keyPairs
}
*/
