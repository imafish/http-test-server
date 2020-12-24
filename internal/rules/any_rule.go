package rules

type anyRule struct {
}

func (r *anyRule) Match(value interface{}, variables map[string]*Variable) (bool, map[string]*Variable, error) {
	return true, variables, nil
}
