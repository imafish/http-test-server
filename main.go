package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/imafish/http-test-server/internal/config"
	"github.com/imafish/http-test-server/internal/handler"
	"github.com/imafish/http-test-server/internal/rules"

	"github.com/fsnotify/fsnotify"
)

func main() {
	configPath := flag.String("c", "", "path to config file. manditory")
	autoReload := flag.Bool("autoreload", false, "relaod config file is content is changed. IMPORTANT: Only rules are reloaded.")
	flag.Parse()

	if *configPath == "" {
		usage()
	}

	config, err := config.LoadConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config file, err: %s", err.Error())
	}

	compiledRules, err := preprocessConfig(config)
	if err != nil {
		log.Fatalf("Failed to verify config object, err: %s", err.Error())
	}

	mtx := sync.Mutex{}
	handler := &handler.RequestHandler{
		Rules: &compiledRules,
		Mtx:   &mtx,
	}

	serverCount := len(config.Servers)
	var wg sync.WaitGroup
	wg.Add(serverCount)

	if *autoReload {
		watchConfigFile(*configPath, &compiledRules, &mtx, &wg)
		wg.Add(1)
	}

	for _, server := range config.Servers {
		go serverFunc(server, handler, &wg)
	}

	wg.Wait()
}

func watchConfigFile(configPath string, rules *[]*rules.CompiledRule, mtx *sync.Mutex, wg *sync.WaitGroup) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to initialize file watcher: %s", err.Error())
		return
	}

	err = watcher.Add(configPath)
	if err != nil {
		log.Printf("Failed to watch for config file: %s", err.Error())
		return
	}

	log.Printf("Starting to watch for config file change...")

	go func() {
		wg.Add(1)
		defer wg.Done()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Printf("\n------- ------- -------")
					log.Printf("config file changed, reloading...")
					config, err := config.LoadConfigFromFile(configPath)
					if err != nil {
						log.Printf("Failed to load config file, err: %s", err.Error())
						continue
					}

					compiledRules, err := preprocessConfig(config)
					if err != nil {
						log.Printf("Failed to verify config object, err: %s", err.Error())
						continue
					}

					mtx.Lock()
					*rules = compiledRules
					mtx.Unlock()

					log.Printf("config file reloaded.")
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("erro watching config file: %s", err)
			}
		}
	}()
}

func serverFunc(server config.ServerConfig, handler http.Handler, wg *sync.WaitGroup) {
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

// preprocessConfig verifies whether manditory fields exists in config object then
// fills missing fields with default value.
// Also, it compiles plain Rule object into CompiledRule, complaining any error found during the process
func preprocessConfig(config *config.Config) ([]*rules.CompiledRule, error) {
	if len(config.Servers) < 1 {
		return nil, fmt.Errorf("server count must be greater than 1")
	}

	for _, server := range config.Servers {
		if (server.CertFile != "" && server.KeyFile == "") || (server.KeyFile != "" && server.CertFile == "") {
			return nil, fmt.Errorf("server.CertFile and server.KeyFile must come in pair")
		}
	}

	compiledRules := make([]*rules.CompiledRule, len(config.Rules))
	for i, r := range config.Rules {
		compiledRule, err := rules.CompileRule(r)
		if err != nil {
			return nil, err
		}

		compiledRules[i] = compiledRule
	}

	return compiledRules, nil
}
