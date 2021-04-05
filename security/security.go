package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"
)

var SecConf = new(SecConfig)

type SecConfig struct {
	Key	string
}

func readSecConfig() (*SecConfig, error) {
	yamlFile, err := ioutil.ReadFile("/root/anvil/config/test_config.yaml")
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, SecConf)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }

        return SecConf, nil
}

func EncData(mytext string) string {
    SecConf, err := readSecConfig()
    if err != nil {
	    log.Fatalln("Unable to read security configuration")
    }
    text := []byte("My Super Secret Code Stuff")
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
    return string(gcm.Seal(nonce, nonce, text, nil))
}

func DecData(encString string) string {
    key := []byte(SecConf.Key)
    ciphertext := []byte(encString)
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
    return (string(plaintext))
}
