package main

import (
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type NodeConfig struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	// Ip       string `yaml:"ip"`
	Port     uint16 `yaml:"port"`
	WsPort   uint16 `yaml:"ws_port"`
	Sequence int    `yaml: "sequence"`
}

type Config struct {
	Nodes []*NodeConfig `yaml: "nodes"`
}

func loadConfig(filepath string) *Config {
	log.Printf("init conf from file %v", filepath)
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	log.Println("yamlFile:", string(yamlFile))

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

type Broadcaster interface {
	BroadcastData(v []byte) error
}
