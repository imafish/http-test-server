package rules

import "github.com/imafish/http-test-server/internal/config"

// CompiledRule is compiled from config.Rule.
// Errors are caught and thrown during compilation.
type CompiledRule struct {
	Request  CompiledRequestRule
	Response config.ResponseRule
	Name     string
}

// CompiledRequestRule is the compiled version of config.RequestRule
// Errors are caught and thrown during compilation.
type CompiledRequestRule struct {
	path    string
	headers []config.HeaderRule
	method  string
	body    BodyRule
}

// BodyRule interface is the for matching request body.
type BodyRule interface {
	Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error)
}
