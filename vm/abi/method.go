package abi

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"strings"
)

// Method represents a callable given a `Names` and whether the method is a constant.
// If the method is `Const` no transaction needs to be created for this
// particular Method call. It can easily be simulated using a local VM.
// For example a `Balance()` method only needs to retrieve something
// from the storage and therefor requires no Tx to be send to the
// network. A method such as `Transact` does require a Tx and thus will
// be flagged `true`.
// Input specifies the required input parameters for this gives method.
type Method struct {
	Name    string
	id      []byte
	Inputs  Arguments
	Outputs Arguments
}

func newMethod(name string, inputs, outputs Arguments) Method {
	m := Method{Name: name, Inputs: inputs, Outputs: outputs}
	m.id = types.DataHash([]byte(m.Sig())).Bytes()[:4]
	return m
}

// Sig returns the methods string signature according to the ABI spec.
//
// Example
//
//     function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (method Method) Sig() string {
	types := make([]string, len(method.Inputs))
	for i, input := range method.Inputs {
		types[i] = input.Type.String()
	}
	return fmt.Sprintf("%v(%v)", method.Name, strings.Join(types, ","))
}

func (method Method) String() string {
	inputs := make([]string, len(method.Inputs))
	for i, input := range method.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	return fmt.Sprintf("onMessage %v(%v)", method.Name, strings.Join(inputs, ", "))
}

func (method Method) Id() []byte {
	return method.id
}
