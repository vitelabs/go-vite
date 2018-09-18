package abi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// The ABIContract holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABIContract struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
	Variables   map[string]Variable
}

// JSONToABIContract returns a parsed ABI interface and error if it failed.
func JSONToABIContract(reader io.Reader) (ABIContract, error) {
	dec := json.NewDecoder(reader)

	var abi ABIContract
	if err := dec.Decode(&abi); err != nil {
		return ABIContract{}, err
	}

	return abi, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABIContract) PackMethod(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	if name == "" {
		// constructor
		arguments, err := abi.Constructor.Inputs.Pack(args...)
		if err != nil {
			return nil, err
		}
		return arguments, nil

	}
	method, exist := abi.Methods[name]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", name)
	}

	arguments, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	return append(method.Id(), arguments...), nil
}

func (abi ABIContract) PackVariable(name string, args ...interface{}) ([]byte, error) {
	variable, exist := abi.Variables[name]
	if !exist {
		return nil, fmt.Errorf("varible '%s' not found", name)
	}

	arguments, err := variable.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	return arguments, nil
}

// UnpackMethod output in v according to the abi specification
func (abi ABIContract) UnpackMethod(v interface{}, name string, output []byte) (err error) {
	if len(output) <= 4 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}
	if method, err := abi.MethodById(output[0:4]); err == nil && method.Name == name {
		return method.Inputs.Unpack(v, output[4:])
	}
	return fmt.Errorf("abi: could not locate named method")
}

// UnpackEvent output in v according to the abi specification
func (abi ABIContract) UnpackEvent(v interface{}, name string, output []byte) (err error) {
	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}
	if event, ok := abi.Events[name]; ok {
		return event.Inputs.Unpack(v, output)
	}
	return fmt.Errorf("abi: could not locate named event")
}

func (abi ABIContract) UnpackVariable(v interface{}, name string, output []byte) (err error) {
	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}
	if variable, ok := abi.Variables[name]; ok {
		return variable.Inputs.Unpack(v, output)
	}
	return fmt.Errorf("abi: could not locate named variable")
}

// UnmarshalJSON implements json.Unmarshaler interface
func (abi *ABIContract) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type      string
		Name      string
		Constant  bool
		Anonymous bool
		Inputs    []Argument
		Outputs   []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	abi.Variables = make(map[string]Variable)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
			// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = Method{
				Name:   field.Name,
				Const:  field.Constant,
				Inputs: field.Inputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:      field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		case "variable":
			if len(field.Inputs) == 0 {
				return fmt.Errorf("abi: could not unmarshal empty variable inputs")
			}
			abi.Variables[field.Name] = Variable{
				Name:   field.Name,
				Inputs: field.Inputs,
			}
		}
	}
	return nil
}

// MethodById looks up a method by the 4-byte id
// returns nil if none found
func (abi *ABIContract) MethodById(sigdata []byte) (*Method, error) {
	for _, method := range abi.Methods {
		if bytes.Equal(method.Id(), sigdata[:4]) {
			return &method, nil
		}
	}
	return nil, fmt.Errorf("no method with id: %#x", sigdata[:4])
}
