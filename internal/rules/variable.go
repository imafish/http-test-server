package rules

import (
	"fmt"
)

// Variable represents a variable when parsing request path or request body
type Variable struct {
	name  string
	vType VariableType
	value interface{}
}

// VariableType represents type of the variable
type VariableType int32

const (
	vtInt VariableType = iota
	vtString
	vtFloat
)

// GetValue returns variable's value, the returned object type is based on variable's type
func (v *Variable) GetValue() (interface{}, error) {
	var value interface{}
	if v.vType == vtInt {
		if intValue, ok := v.value.(int); ok {
			value = intValue
		} else {
			return nil, fmt.Errorf("variable type is int, but value isn't")
		}

	} else if v.vType == vtFloat {
		if floatValue, ok := v.value.(float64); ok {
			value = floatValue
		} else {
			return nil, fmt.Errorf("variable type is float, but value isn't")
		}

	} else if v.vType == vtString {
		if strValue, ok := v.value.(string); ok {
			value = strValue
		} else {
			return nil, fmt.Errorf("variable type is string, but value isn't")
		}
	} else {
		return nil, fmt.Errorf("unexpected variable type")
	}

	return value, nil
}
