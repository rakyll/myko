package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/mykodev/myko/compactor"
	"github.com/mykodev/myko/config"
	pb "github.com/mykodev/myko/proto"
	"github.com/mykodev/myko/server"
)

var configFile string

func main() {
	ctx := context.Background()

	flag.StringVar(&configFile, "config", "", "")
	flag.Parse()

	var cfg config.Config
	if configFile == "" {
		cfg = config.DefaultConfig()
	} else {
		var err error
		cfg, err = config.Open(configFile)
		if err != nil {
			log.Fatalf("Failed to open and parse config file: %v", err)
		}
	}

	service, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create a server: %v", err)
	}

	log.Printf("Starting the myko server at %q...", cfg.Listen)
	server := pb.NewServiceServer(service, nil)
	log.Fatal(http.ListenAndServe(cfg.Listen, server))
}
