package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"sync"

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

	serverCount := len(config.Servers)
	var wg sync.WaitGroup
	wg.Add(serverCount)

	for _, server := range config.Servers {
		go serverFunc(server, handler, &wg)
	}

	wg.Wait()
}

func serverFunc(server common.ServerConfig, handler http.Handler, wg *sync.WaitGroup) {
	defer wg.Done()

	if server.KeyFile != "" {
		log.Printf("HTTPs server listening on %s, key file: %s, cert file: %s", server.Addr, server.KeyFile, server.CertFile)
		log.Fatal(http.ListenAndServeTLS(server.Addr, server.CertFile, server.KeyFile, handler))

	} else {
		log.Printf("HTTP server listening on %s", server.Addr)
		log.Fatal(http.ListenAndServe(server.Addr, handler))
	}
}

func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}
