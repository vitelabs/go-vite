package types

import "strings"

type Addresses []Address

func (as Addresses) String() string {
	var output []string
	for _, a := range as {
		output = append(output, a.String())
	}
	return strings.Join(output, "\n")
}
