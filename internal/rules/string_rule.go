package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type stringRule struct {
	variables   []*Variable
	regex       *regexp.Regexp
	singleMatch bool
}

func (r *stringRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {
	var str string

	if r.singleMatch {
		f, ok := value.(float64)
		if !ok {
			return false, nil, nil
		}
		str = fmt.Sprintf("%f", f)
		str = strings.TrimRight(str, ".0")
	} else {
		s, ok := value.(string)
		if !ok {
			return false, nil, nil
		}
		str = s
	}

	matches := r.regex.FindAllStringSubmatch(str, -1)
	if matches == nil {
		return false, nil, nil
	}

	// only process the first match
	submatches := matches[0]
	for i, sm := range submatches[1:] {

		// Let's make a copy here, so the matched variable does alter variable objects in Rule
		variable := *r.variables[i]

		switch variable.vType {
		case vtInt:
			variable.value, _ = strconv.Atoi(sm)

		case vtString:
			variable.value = sm

		case vtFloat:
			variable.value, _ = strconv.ParseFloat(sm, 64)
		}

		variables[variable.name] = &variable
	}

	return true, variables, nil
}
