package rules

import (
	"fmt"
	"regexp"

	"github.com/imafish/http-test-server/internal/config"
)

// CompileRule compiled plain Rule object generated from a config file into compiled rules so it simplifies also decouple rule matching
// Also it finds any errors in the plain Rule object and returns an error object
// Currently only request body rule is compiled.
func CompileRule(rule config.Rule) (*CompiledRule, error) {
	bodyRule, err := compileBodyRule(rule.Request.Body)
	if err != nil {
		return nil, err
	}

	compiled := &CompiledRule{
		Request: CompiledRequestRule{
			path:    rule.Request.Path,
			headers: rule.Request.Headers,
			method:  rule.Request.Method,
			body:    bodyRule,
		},
		Response: rule.Response,
		Name:     rule.Name,
	}

	return compiled, nil
}

func compileBodyRule(bodyRule config.RequestBodyRule) (BodyRule, error) {
	if bodyRule.Value == nil && bodyRule.MatchRule == "" {
		return nil, nil
	}

	var strict bool
	if bodyRule.MatchRule == "loose" || bodyRule.MatchRule == "" {
		strict = false
	} else if bodyRule.MatchRule == "strict" {
		strict = true
	} else {
		return nil, fmt.Errorf("bodyRulecompiledRule.MatchRule must be one of 'loose' and 'strict'")
	}

	variableNames := make(map[string]bool)

	return compileObject(bodyRule.Value, strict, variableNames)
}

func compileObject(value interface{}, strict bool, variableNames map[string]bool) (BodyRule, error) {
	switch e := value.(type) {
	case string:
		return compileStringRule(e, strict, variableNames)

	case float64:
		compiled := &numberRule{
			expected: e,
		}
		return compiled, nil

	case int:
		compiled := &numberRule{
			expected: float64(e),
		}
		return compiled, nil

	case map[interface{}]interface{}:
		return compileMap(e, strict, variableNames)

	case []interface{}:
		return compileSlice(e, strict, variableNames)

	default:
		return nil, fmt.Errorf("Encountered invalid bodyRule.Value type")
	}
}

func compileSlice(rules []interface{}, strict bool, variableNames map[string]bool) (BodyRule, error) {

	compiledRules := make([]BodyRule, len(rules))

	for i, v := range rules {
		compiledRule, err := compileObject(v, strict, variableNames)
		if err != nil {
			return nil, err
		}

		compiledRules[i] = compiledRule
	}

	sr := &sliceRule{
		subRules: compiledRules,
	}

	return sr, nil
}

func compileMap(rules map[interface{}]interface{}, strict bool, variableNames map[string]bool) (BodyRule, error) {

	compiledRules := make(map[string]BodyRule)

	for k, v := range rules {
		kString, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("key of map object in Rule must be of type string")
		}

		compiledRule, err := compileObject(v, strict, variableNames)
		if err != nil {
			return nil, err
		}

		compiledRules[kString] = compiledRule
	}

	mr := &mapRule{
		strict:   strict,
		subRules: compiledRules,
	}

	return mr, nil
}

var matchVariableRegex = regexp.MustCompile(`{{(\w+),(\w+)}}`)

func compileStringRule(rule string, strict bool, variableNames map[string]bool) (BodyRule, error) {

	matches := matchVariableRegex.FindAllStringSubmatchIndex(rule, -1)
	startIndex := 0
	extractedVariables := make([]*Variable, len(matches))

	singleMatch := false
	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(rule) {
		singleMatch = true
	}

	var regexString string
	for i, matchIndex := range matches {
		leadingPart := rule[startIndex:matchIndex[0]]
		if strict {
			leadingPart = escapeRegexSpecialCharacters(leadingPart)
		}
		regexString += leadingPart

		variableName := rule[matchIndex[2]:matchIndex[3]]
		_, ok := variableNames[variableName]
		if ok {
			return nil, fmt.Errorf("multiple variable with name %s found in rules", variableName)
		}
		variableNames[variableName] = true

		variableTypeStr := rule[matchIndex[4]:matchIndex[5]]
		var rulePart string
		var vType VariableType
		switch variableTypeStr {
		case "int":
			rulePart = `([-+]?\d+)`
			vType = vtInt

		case "string":
			rulePart = `(.+)`
			vType = vtString

		case "float":
			rulePart = `([-+]?[0-9]*\.?[0-9]+)`
			vType = vtFloat

		default:
			return nil, fmt.Errorf("Invalid variable type %s found in rules", variableTypeStr)
		}
		regexString += rulePart
		extractedVariables[i] = &Variable{
			name:  variableName,
			vType: vType,
		}

		startIndex = matchIndex[1]
	}

	endPart := rule[startIndex:]
	if strict {
		endPart = escapeRegexSpecialCharacters(endPart)
	}
	regexString += endPart

	if strict {
		regexString = "^" + regexString + "$"
	}

	regex, err := regexp.Compile(regexString)
	if err != nil {
		return nil, fmt.Errorf("failed to compile string into regex. err: %s", err.Error())
	}

	compiled := &stringRule{
		regex:       regex,
		variables:   extractedVariables,
		singleMatch: singleMatch,
	}

	return compiled, nil
}

var escapeRegex = regexp.MustCompile(`([\.\*\[\]\(\)\\])`)

func escapeRegexSpecialCharacters(unescaped string) string {
	return escapeRegex.ReplaceAllString(unescaped, `\$1`)
}
