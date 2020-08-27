package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/imafish/http-test-server/internal/config"
)

// FindMatchingRule returns the first matching rule from slices of rule
func FindMatchingRule(rules []*CompiledRule, request *http.Request) (*CompiledRule, map[string]*Variable, error) {
	var matchedRule *CompiledRule
	variables := make(map[string]*Variable)

	for _, r := range rules {
		requestRule := r.Request
		log.Printf("RULE: path: %s, method %s\n", requestRule.path, requestRule.method)

		match := (requestRule.method == request.Method)
		if !match {
			log.Printf("method not match, expect: %s, got %s\n", requestRule.method, request.Method)
			continue
		}

		match, err := matchPath(requestRule.path, request.RequestURI)
		if err != nil {
			return nil, nil, err
		}
		if !match {
			log.Printf("path not match, expect: %s, got %s\n", requestRule.path, request.RequestURI)
			continue
		}

		match, err = matchHeaders(requestRule.headers, request.Header)
		if err != nil {
			return nil, nil, err
		}
		if !match {
			log.Printf("header not match\n")
			continue
		}

		match, variables, err = matchBody(requestRule.body, request.Body)
		if err != nil {
			return nil, nil, err
		}
		if !match {
			log.Printf("body not match")
			continue
		}

		matchedRule = r
		break
	}

	return matchedRule, variables, nil
}

func matchPath(path string, requestPath string) (bool, error) {
	// TODO @XG this method should return a 'context' map in future version.
	// a context map stores path params extracted from request URI
	// e.g.: /api/book/{id}/title, /api/book/2/title => [id]=2

	ruleSplits := strings.Split(strings.TrimLeft(path, "/"), "/")
	requestSplits := strings.Split(strings.TrimLeft(requestPath, "/"), "/")

	if len(ruleSplits) != len(requestSplits) {
		return false, nil
	}

	for i, rs := range ruleSplits {
		regx, err := regexp.Compile(rs)
		if err != nil {
			// TODO @XG move this erro checking to rule compilation phase
			return false, fmt.Errorf("Failed to compile regex from %s, err: %s", rs, err.Error())
		}

		pathPart := requestSplits[i]
		match := regx.MatchString(pathPart)
		if !match {
			return false, nil
		}
	}

	return true, nil
}

func matchHeaders(headerRules []config.HeaderRule, requestHeader http.Header) (bool, error) {

	requestHeaderStrings := make([]string, 0)
	for k, v := range requestHeader {
		for _, hs := range v {
			requestHeaderStrings = append(requestHeaderStrings, fmt.Sprintf("%s: %s", k, hs))
		}
	}

	for _, hr := range headerRules {
		match := false

		if hr.Include == "" && hr.Not == "" {
			return false, fmt.Errorf("header rule must have one of Include and Not clause")
		}

		if hr.Include != "" {
			regx, err := regexp.Compile(hr.Include)
			if err != nil {
				return false, fmt.Errorf("Failed to compile regex from %s, err: %s", hr.Include, err.Error())
			}

			for _, hs := range requestHeaderStrings {
				match = regx.MatchString(hs)
				if match {
					break
				}
			}
			if !match {
				return false, nil
			}

		} else if hr.Not != "" {
			regx, err := regexp.Compile(hr.Not)
			if err != nil {
				return false, fmt.Errorf("Failed to compile regex from %s, err: %s", hr.Not, err.Error())
			}

			for _, hs := range requestHeaderStrings {
				match = regx.MatchString(hs)
				if match {
					return false, nil
				}
			}

		} else {
			return false, fmt.Errorf("header rule should only have one of Include and Not clause")
		}

	}

	return true, nil
}

func matchBody(bodyRule BodyRule, requestBody io.ReadCloser) (bool, map[string]*Variable, error) {

	if bodyRule == nil {
		return true, nil, nil
	}

	// TODO @XG this variables should be passed in from method parameter, as matchPath also generates variables.
	variables := make(map[string]*Variable)

	bytes, err := ioutil.ReadAll(requestBody)
	bodyObj := make(map[string]interface{})
	err = json.Unmarshal(bytes, &bodyObj)
	if err == nil {
		return bodyRule.Match(bodyObj, variables)
	}
	bodySlice := make([]interface{}, 0)
	err = json.Unmarshal(bytes, &bodySlice)
	if err == nil {
		return bodyRule.Match(bodySlice, variables)
	}
	var bodyNumber float64
	err = json.Unmarshal(bytes, &bodyNumber)
	if err == nil {
		return bodyRule.Match(bodyNumber, variables)
	}
	return bodyRule.Match(string(bytes), variables)
}

func matchString(expected string, actual string, strict bool) (bool, error) {
	// find all possible variable definitions in rule

	if strict {
		return actual == expected, nil
	}

	regex, err := regexp.Compile(expected)
	if err != nil {
		err = fmt.Errorf("Failed to compile regex from rule, err: %s", err.Error())
		return false, err
	}
	return regex.MatchString(actual), nil
}

func matchObject(expected interface{}, actual interface{}, strict bool) (bool, error) {

	_, ok1 := expected.(map[interface{}]interface{})
	_, ok2 := actual.(map[string]interface{})
	_, ok3 := expected.(int)
	_, ok4 := actual.(float64)
	if (ok1 && ok2) || (ok3 && ok4) {
		// YAML parse 123 as int while JSON parse 123 as float64;
		// YAML parse object as map[interface{}]interface{} while JSON parse object as map[string]interface{}
		// So should treat the two above scenarios specially.
	} else if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return false, nil
	}

	var equal bool
	var err error

	switch e := expected.(type) {
	case int:
		aFloat, ok := actual.(float64)
		var aInt int
		if ok {
			aInt = int(aFloat)
		} else {
			aInt = actual.(int)
		}
		equal = (aInt == e)

	case float64:
		a := actual.(float64)
		equal = (a == e)

	case string:
		a := actual.(string)
		equal, err = matchString(e, a, strict)

	case map[interface{}]interface{}:
		a := actual.(map[string]interface{})
		equal, err = matchMap(e, a, strict)

	case []interface{}:
		a := actual.([]interface{})
		equal, err = matchSlice(e, a, strict)

	default:
		equal, err = false, fmt.Errorf("Unexpected type in YAML map object")
	}

	return equal, err
}

func matchMap(expected map[interface{}]interface{}, actual map[string]interface{}, strict bool) (bool, error) {

	if strict && len(expected) != len(actual) {
		return false, nil
	}

	for key, value := range expected {
		keyString, ok := key.(string)
		if !ok {
			return false, fmt.Errorf("key in YAML map should be of type string")
		}

		bodyValue, ok := actual[keyString]
		if !ok {
			return false, nil
		}

		equal, err := matchObject(value, bodyValue, strict)
		if !equal || err != nil {
			return equal, err
		}
	}

	return true, nil
}

func matchSlice(expected, actual []interface{}, strict bool) (bool, error) {
	if len(expected) != len(actual) {
		return false, nil
	}

	for i, e := range expected {
		a := actual[i]
		equal, err := matchObject(e, a, strict)
		if !equal || err != nil {
			return equal, err
		}
	}

	return true, nil
}
