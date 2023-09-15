package main

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config struct {
	// map of challenge name to manifest in yaml. supports multiple objects per file delimited with ---
	Challenges  map[string]string `json:"challenges"`
	ListenAddr  string            `json:"listenAddr"`
	ResetDB     bool              `json:"resetDB"`
	InstanceTTL string            `json:"instanceTTL"`
	UseTemplate string            `json:"useTemplate"`
}

func loadConfig() (*Config, error) {
	confb, err := os.ReadFile("/config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("when reading config file:\n\t%v", err)
	}
	conf := Config{
		ListenAddr:  ":8080",
		ResetDB:     false,
		InstanceTTL: "10m",
	}
	err = yaml.Unmarshal(confb, &conf)
	if err != nil {
		return nil, fmt.Errorf("when parsing config file:\n\t%v", err)
	}
	return &conf, nil
}
