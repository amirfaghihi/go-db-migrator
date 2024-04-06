package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	MongoDB struct {
		URI string `yaml:"uri"`
	} `yaml:"mongodb"`
}

var cfg Config

// LoadConfig reads the configuration from a file
func LoadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return err
	}

	return nil
}

// GetMongoURI returns the MongoDB connection URI
func GetMongoURI() string {
	return cfg.MongoDB.URI
}
