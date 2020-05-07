package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	yamlFile, err := ioutil.ReadFile("../../etc/default-simple.yaml")

	log.Println("yamlFile:", yamlFile)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
	}
	conf := new(Node)

	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	yaml1, _ := json.Marshal(conf)
	log.Println("conf: \n", string(yaml1))
}
