package abi

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"strings"
)

// Event is an event potentially triggered by the VM's LOG mechanism. The Event
// holds type information (inputs) about the yielded output. Anonymous events
// don't get the signature canonical representation as the first LOG topic.
type Event struct {
	Name      string
	Anonymous bool
	Inputs    Arguments
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
