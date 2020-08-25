package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/imafish/http-test-server/internal/common"
)

func main() {
	configPath := flag.String("c", "", "path to config file. manditory")
	flag.Parse()

	if *configPath == "" {
		usage()
	}

	config, err := common.LoadConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config file, err: %s", err.Error())
	}

	err = common.PreprocessConfig(config)
	if err != nil {
		log.Fatalf("Failed to verify config object, err: %s", err.Error())
	}

	handler := &common.RequestHandler{
		Rules: config.Rules,
	}

	log.Printf("HTTP server listening on %s", config.Server.Addr)
	log.Fatal(http.ListenAndServe(config.Server.Addr, handler))
}

func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}
