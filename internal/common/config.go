package common

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the config of this application
type Config struct {
	Servers []ServerConfig
	Rules   []Rule
}

// ServerConfig represents the config for the HTTP(S) server
type ServerConfig struct {
	Addr     string
	CertFile string `yaml:"cert_file,omitempty"` // path to the cert file
	KeyFile  string `yaml:"key_file,omitempty"`  // path to the key file
}

// Rule represents a rule
type Rule struct {
	Request  RequestRule
	Response ResponseRule
}

// RequestRule represents request rule
type RequestRule struct {
	Path    string
	Headers []HeaderRule
	Method  string
	Body    RequestBodyRule
}

// HeaderRule represents header rule
type HeaderRule struct {
	Include string
	Not     string
}

// RequestBodyRule represents the matching rule for request body
type RequestBodyRule struct {
	MatchRule string `yaml:"match_rule"`
	Value     interface{}
}

// ResponseRule represents response rule
type ResponseRule struct {
	Status  int
	Headers []string
	File    string
	Body    interface{}
}

// LoadConfigFromFile loads the config from a YAML file
func LoadConfigFromFile(configPath string) (*Config, error) {
	config := Config{}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.SetStrict(false)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// PreprocessConfig verifies whether manditory fields exists in config object then
// fills missing fields with default value.
func PreprocessConfig(config *Config) error {
	if len(config.Servers) < 1 {
		return fmt.Errorf("server count must be greater than 1")
	}

	for _, server := range config.Servers {
		if (server.CertFile != "" && server.KeyFile == "") || (server.KeyFile != "" && server.CertFile == "") {
			return fmt.Errorf("server.CertFile and server.KeyFile must come in pair")
		}
	}

	// TODO @XG also validates rules

	return nil
}
