package rules

type mapRule struct {
	strict   bool
	subRules map[string]BodyRule
}

func (r *mapRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {
	valueMap, ok := value.(map[string]interface{})
	if !ok {
		return false, variables, nil
	}

	ruleTracking := make(map[string]bool)
	for k, v := range r.subRules {
		_, ok := v.(*anyRule)
		ruleTracking[k] = ok
	}

	for k, v := range valueMap {
		subRule := r.subRules[k]
		if subRule == nil {
			return false, variables, nil
		}
		ruleTracking[k] = true

		var isMatch bool
		var err error
		isMatch, variables, err = subRule.Match(v, variables)
		if err != nil {
			return false, nil, err
		}
		if !isMatch {
			return false, variables, nil
		}
	}

	if r.strict {
		for _, v := range ruleTracking {
			if !v {
				return false, variables, nil
			}
		}
	}

	return true, variables, nil
}
