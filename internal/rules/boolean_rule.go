package rules

type booleanRule struct {
	expected bool
}

func (r *booleanRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {
	actual, ok := value.(bool)
	if !ok {
		return false, variables, nil
	}

	return r.expected == actual, variables, nil
}
