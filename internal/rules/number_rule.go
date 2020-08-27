package rules

type numberRule struct {
	expected float64
}

func (r *numberRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {
	actual, ok := value.(float64)
	if !ok {
		return false, nil, nil
	}

	return r.expected == actual, variables, nil
}
