package rules

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/imafish/http-test-server/internal/config"
)

// FindMatchingRule returns the first matching rule from slices of rule
func FindMatchingRule(rules *[]*CompiledRule, request *http.Request) (*CompiledRule, map[string]*Variable, error) {
	var matchedRule *CompiledRule
	variables := make(map[string]*Variable)

	bytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, nil, err
	}

	for _, r := range *rules {
		requestRule := r.Request

		match := (requestRule.method == request.Method)
		if !match {
			continue
		}

		match, err := matchPath(requestRule.path, request.RequestURI)
		if err != nil {
			return nil, nil, err
		}
		if !match {
			continue
		}

		match, err = matchHeaders(requestRule.headers, request.Header)
		if err != nil {
			return nil, nil, err
		}
		if !match {
			continue
		}

		match, variables, err = matchBody(requestRule.body, bytes)
		if err != nil {
			return nil, nil, err
		}
		if !match {
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

func matchBody(bodyRule BodyRule, bytes []byte) (bool, map[string]*Variable, error) {

	if bodyRule == nil {
		return true, nil, nil
	}

	// TODO @XG this variables should be passed in from method parameter, as matchPath also generates variables.
	variables := make(map[string]*Variable)

	bodyObj := make(map[string]interface{})
	err := json.Unmarshal(bytes, &bodyObj)
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
