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

	server := config.Server
	if server.KeyFile != "" {
		log.Printf("HTTPs server listening on %s, key file: %s, cert file: %s", server.Addr, server.KeyFile, server.CertFile)
		log.Fatal(http.ListenAndServeTLS(server.Addr, server.CertFile, server.KeyFile, handler))

	} else {
		log.Printf("HTTP server listening on %s", config.Server.Addr)
		log.Fatal(http.ListenAndServe(config.Server.Addr, handler))
	}
}

func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}
