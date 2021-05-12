package acl

import (
	"math/rand"
	"time"
	b64 "encoding/base64"
        //"fmt"
        "io/ioutil"
	"log"

        "gopkg.in/yaml.v2"
)

var aclList []ACLEntry

type ACLEntry struct {
	Name		string		`yaml:"name,omitempty"`
	TokenValue	string
	CreationTime	time.Time
	ExpirationTime	time.Time
	ServiceList	[]string	`yaml:"services,omitempty"`
}

func ACLIngest(filepath string) ([]ACLEntry, error) {
	var tempList []ACLEntry
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("Read file error #%v", err)
	}
	err = yaml.Unmarshal(yamlFile, &tempList)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	for _,ele := range tempList {
		tokVal := StringWithCharset(64, charset)
		createTime := time.Now()
		expTime := time.Now().Add(24*time.Hour)
		aclList = append(aclList, ACLEntry{Name: ele.Name, TokenValue: tokVal, CreationTime: createTime,
				ExpirationTime: expTime, ServiceList: ele.ServiceList})
	}
	return aclList, nil
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
  rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return b64.StdEncoding.EncodeToString(b)
}
