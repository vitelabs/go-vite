package types

import "errors"

var (
	ErrJsonNotString = errors.New("not valid string")
)

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

func trimLeftRightQuotation(input []byte) []byte {
	if isString(input) {
		return input[1 : len(input)-1]
	}
	return []byte{}
}
