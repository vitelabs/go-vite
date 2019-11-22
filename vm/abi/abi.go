package abi

import (
	"bytes"
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"io"
)

// The ABIContract holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABIContract struct {
	Constructor Method
	Methods     map[string]Method
	Callbacks   map[string]Method
	OffChains   map[string]Method
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

func (abi ABIContract) PackMethod(name string, args ...interface{}) ([]byte, error) {
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
		return nil, errMethodNotFound(name)
	}
	return abi.packMethod(method, args...)
}

func (abi ABIContract) PackCallback(name string, args ...interface{}) ([]byte, error) {
	callbackName := getCallBackName(name)
	method, exist := abi.Callbacks[callbackName]
	if !exist {
		return nil, errCallbackNotFound(name)
	}
	return abi.packMethod(method, args...)
}

func (abi ABIContract) PackOffChain(name string, args ...interface{}) ([]byte, error) {
	method, exist := abi.OffChains[name]
	if !exist {
		return nil, errOffchainNotFound(name)
	}
	return abi.packMethod(method, args...)
}

func (abi ABIContract) PackVariable(name string, args ...interface{}) ([]byte, error) {
	variable, exist := abi.Variables[name]
	if !exist {
		return nil, errVariableNotFound(name)
	}

	return variable.Inputs.Pack(args...)
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABIContract) packMethod(method Method, args ...interface{}) ([]byte, error) {
	arguments, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	return append(method.Id(), arguments...), nil
}

func (abi ABIContract) PackEvent(name string, args ...interface{}) (topics []types.Hash, data []byte, err error) {
	e, exist := abi.Events[name]
	if !exist {
		return nil, nil, errEventNotFound(name)
	}
	return e.Pack(args...)
}

// UnpackMethod output in v according to the abi specification
func (abi ABIContract) UnpackMethod(v interface{}, name string, input []byte) (err error) {
	if len(input) <= 4 {
		return errEmptyInput
	}
	if method, err := abi.MethodById(input[0:4]); err == nil && method.Name == name {
		return method.Inputs.Unpack(v, input[4:])
	}
	return errCouldNotLocateNamedMethod
}

// DirectUnpackMethodInput output in param list according to the abi specification
func (abi ABIContract) DirectUnpackMethodInput(name string, input []byte) ([]interface{}, error) {
	if len(input) <= 4 {
		return nil, errEmptyInput
	}
	if method, err := abi.MethodById(input[0:4]); err == nil && method.Name == name {
		return method.Inputs.DirectUnpack(input[4:])
	}
	return nil, errCouldNotLocateNamedMethod
}

// DirectUnpackOffchainOutput output in param list according to the abi specification
func (abi ABIContract) DirectUnpackOffchainOutput(name string, output []byte) ([]interface{}, error) {
	if offchain, ok := abi.OffChains[name]; ok {
		return offchain.Outputs.DirectUnpack(output)
	}
	return nil, errOffchainNotFound(name)
}

// UnpackEvent output in v according to the abi specification
func (abi ABIContract) UnpackEvent(v interface{}, name string, input []byte) (err error) {
	if len(input) == 0 {
		return errEmptyInput
	}
	if event, ok := abi.Events[name]; ok {
		return event.NonIndexedInputs.Unpack(v, input)
	}
	return errCouldNotLocateNamedEvent
}

// DirectUnpackEvent output in param list according to the abi specification
func (abi ABIContract) DirectUnpackEvent(topics []types.Hash, data []byte) (string, []interface{}, error) {
	if len(topics) < 1 {
		return "", nil, errEmptyInput
	}
	for _, event := range abi.Events {
		if event.Id() == topics[0] {
			params, err := event.DirectUnPack(topics, data)
			return event.String(), params, err
		}
	}
	return "", nil, errCouldNotLocateNamedEvent
}

func (abi ABIContract) UnpackVariable(v interface{}, name string, input []byte) (err error) {
	if len(input) == 0 {
		return errEmptyInput
	}
	if variable, ok := abi.Variables[name]; ok {
		return variable.Inputs.Unpack(v, input)
	}
	return errCouldNotLocateNamedVariable
}

// MethodById looks up a method by the 4-byte id
// returns nil if none found
func (abi *ABIContract) MethodById(sigdata []byte) (*Method, error) {
	if len(sigdata) < 4 {
		return nil, errMethodIdNotSpecified
	}
	for _, method := range abi.Methods {
		if bytes.Equal(method.Id(), sigdata[:4]) {
			return &method, nil
		}
	}
	return nil, errNoMethodId(sigdata[:4])
}

// UnmarshalJSON implements json.Unmarshaler interface
func (abi *ABIContract) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type    string
		Name    string
		Inputs  []Argument
		Outputs []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Callbacks = make(map[string]Method)
	abi.OffChains = make(map[string]Method)
	abi.Events = make(map[string]Event)
	abi.Variables = make(map[string]Variable)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = newMethod("", field.Inputs, nil)
			// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = newMethod(field.Name, field.Inputs, nil)
		case "callback":
			name := getCallBackName(field.Name)
			abi.Callbacks[name] = newMethod(name, field.Inputs, nil)
		case "offchain":
			abi.OffChains[field.Name] = newMethod(field.Name, field.Inputs, field.Outputs)
		case "event":
			indexed, nonIndexed := getEventInputs(field.Inputs)
			abi.Events[field.Name] = Event{
				Name:             field.Name,
				Inputs:           field.Inputs,
				IndexedInputs:    indexed,
				NonIndexedInputs: nonIndexed,
			}
		case "variable":
			if len(field.Inputs) == 0 {
				return errInvalidEmptyVariableInput
			}
			abi.Variables[field.Name] = Variable{
				Name:   field.Name,
				Inputs: field.Inputs,
			}
		}
	}
	return nil
}

func getEventInputs(args []Argument) (indexed, nonIndexed []Argument) {
	for _, arg := range args {
		if arg.Indexed {
			indexed = append(indexed, arg)
		} else {
			nonIndexed = append(nonIndexed, arg)
		}
	}
	return
}
func getCallBackName(name string) string {
	return name + "Callback"
}
