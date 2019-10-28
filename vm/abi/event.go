package abi

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"reflect"
	"strings"
)

// Event is an event potentially triggered by the VM's LOG mechanism. The Event
// holds type information (inputs) about the yielded output. Anonymous events
// don't get the signature canonical representation as the first LOG topic.
type Event struct {
	Name             string
	Inputs           Arguments
	IndexedInputs    Arguments
	NonIndexedInputs Arguments
}

func (e Event) String() string {
	inputs := make([]string, len(e.Inputs))
	for i, input := range e.Inputs {
		if input.Indexed {
			inputs[i] = fmt.Sprintf("%v indexed %v", input.Type, input.Name)
		} else {
			inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
		}
	}
	return fmt.Sprintf("event %v(%v)", e.Name, strings.Join(inputs, ", "))
}

// Id returns the canonical representation of the event's signature used by the
// abi definition to identify event names and types.
func (e Event) Id() types.Hash {
	typeList := make([]string, len(e.Inputs))
	i := 0
	for _, input := range e.Inputs {
		typeList[i] = input.Type.String()
		i++
	}
	return types.DataHash([]byte(fmt.Sprintf("%v(%v)", e.Name, strings.Join(typeList, ","))))
}

func (e Event) Pack(args ...interface{}) ([]types.Hash, []byte, error) {
	if len(args) != len(e.Inputs) {
		return nil, nil, errArgLengthMismatch(args, e.Inputs)
	}
	topics := make([]types.Hash, len(e.IndexedInputs)+1)
	topics[0] = e.Id()
	var nonIndexedArgList []interface{}
	topicIndex := 1
	for i := 0; i < len(args); i++ {
		if e.Inputs[i].Indexed {
			topic, err := e.Inputs[i].Type.pack(reflect.ValueOf(args[i]))
			if err != nil {
				return nil, nil, err
			}
			if len(topic) <= types.HashSize {
				topics[topicIndex], _ = types.BytesToHash(helper.LeftPadBytes(topic, types.HashSize))
			} else {
				topics[topicIndex] = types.DataHash(topic)
			}
			topicIndex = topicIndex + 1
		} else {
			nonIndexedArgList = append(nonIndexedArgList, args[i])
		}
	}
	if len(nonIndexedArgList) > 0 {
		d, err := e.NonIndexedInputs.Pack(nonIndexedArgList...)
		if err != nil {
			return nil, nil, err
		}
		return topics, d, nil
	} else {
		return topics, nil, nil
	}
}

func (e Event) DirectUnPack(topics []types.Hash, data []byte) ([]interface{}, error) {
	nonIndexedParams, err := e.NonIndexedInputs.DirectUnpack(data)
	if err != nil {
		return nil, err
	}
	nonIndex := 0
	index := 1
	args := make([]interface{}, 0)
	for _, arg := range e.Inputs {
		if arg.Indexed {
			if arg.Type.T == ArrayTy || arg.Type.T == StringTy || arg.Type.T == SliceTy || arg.Type.T == BytesTy {
				args = append(args, topics[index])
			} else {
				arg, err := toGoType(0, arg.Type, topics[index].Bytes())
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
			index = index + 1
		} else {
			args = append(args, nonIndexedParams[nonIndex])
			nonIndex = nonIndex + 1
		}
	}
	return args, nil
}

func (e *Event) UnmarshalJSON(data []byte) error {
	var fields struct {
		Type    string
		Name    string
		Inputs  []Argument
		Outputs []Argument
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	indexed, nonIndexed := getEventInputs(fields.Inputs)
	e.Name = fields.Name
	e.Inputs = fields.Inputs
	e.IndexedInputs = indexed
	e.NonIndexedInputs = nonIndexed
	return nil
}
