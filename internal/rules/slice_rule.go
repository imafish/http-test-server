package rules

type sliceRule struct {
	subRules []BodyRule
}

func (r *sliceRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {

	sliceValue, ok := value.([]interface{})
	if !ok {
		return false, variables, nil
	}

	if len(sliceValue) != len(r.subRules) {
		return false, variables, nil
	}

	for i, ss := range sliceValue {
		subRule := r.subRules[i]

		var isMatch bool
		var err error
		isMatch, variables, err = subRule.Match(ss, variables)
		if err != nil {
			return false, nil, err
		}
		if !isMatch {
			return false, variables, nil
		}
	}

	return true, variables, nil
}
