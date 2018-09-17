package abi

import (
	"fmt"
	"strings"
)

// Variable is used only in precompiled contracts
type Variable struct {
	Name      string
	Inputs    Arguments
}

func (v Variable) String() string {
	inputs := make([]string, len(v.Inputs))
	for i, input := range v.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	return fmt.Sprintf("struct %v{%v}", v.Name, strings.Join(inputs, "; "))
}
