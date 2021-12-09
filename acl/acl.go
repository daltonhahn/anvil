package acl

import (
	"math/rand"
	"time"
	b64 "encoding/base64"
        "io/ioutil"
	"log"
	"fmt"

        "gopkg.in/yaml.v2"
)

type ACLEntry struct {
	Name		string			`yaml:"name,omitempty"`
	TokenValue	string			`yaml:"val,omitempty"`
	CreationTime	time.Time
	ExpirationTime	time.Time
	TargetService	string	`yaml:"sername,omitempty"`
	ServiceChain	string	`yaml:"serchain,omitempty"`
	//Chains		map[string]ServiceMap	`yaml:"services,omitempty"`
}

func ACLIngest(filepath string) ([]ACLEntry, error) {
	var tempList []ACLEntry
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("Unable to read file for acl ingest")
		log.Printf("Read file error #%v", err)
	}
	err = yaml.Unmarshal(yamlFile, &tempList)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	var retList []ACLEntry
	fmt.Printf("%v\n", tempList)
	/*
	for _,ele := range tempList {
		tokVal := ele.TokenValue
		if len(tokVal) == 0 {
			tokVal = StringWithCharset(64, charset)
		}
		createTime := time.Now()
		expTime := time.Now().Add(24*time.Hour)
		retList = append(retList, ACLEntry{Name: ele.Name, TokenValue: tokVal, CreationTime: createTime,
				ExpirationTime: expTime, ServiceList: ele.ServiceList})
	}
	*/
	return retList, nil
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
