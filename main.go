package main

import (
	"log"
	"os"

	"github.com/amirfaghihi/migrator/config"
	"github.com/amirfaghihi/migrator/mongoClient"
)

func main() {
	configPath, ok := os.LookupEnv("CONFIG_PATH")
	if !ok {
		configPath = "config/config.yml"
	}
	err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize MongoDB client
	mongoClientInstance := mongoClient.Mongo{}
	err = mongoClientInstance.InitClient()
	if err != nil {
		log.Fatal(err)
	}

}
