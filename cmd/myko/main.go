package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/mykodev/myko/config"
	pb "github.com/mykodev/myko/proto"
	"github.com/mykodev/myko/server"
)

var configFile string

func main() {
	flag.StringVar(&configFile, "config", "", "")
	flag.Parse()

	var serverConfig config.Config
	if configFile == "" {
		serverConfig = config.DefaultConfig()
	} else {
		cfg, err := config.Open(configFile)
		if err != nil {
			log.Fatalf("Failed to open and parse config file: %v", err)
		}
		serverConfig = cfg
	}

	service, err := server.New(serverConfig)
	if err != nil {
		log.Fatalf("Failed to create a server: %v", err)
	}

	log.Printf("Starting the myko server at %q...", serverConfig.Listen)
	server := pb.NewServiceServer(service, nil)
	log.Fatal(http.ListenAndServe(serverConfig.Listen, server))
}
