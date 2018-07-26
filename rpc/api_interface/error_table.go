package api_interface

import (
	"encoding/json"
)

const (
	errPassword            = "cipher: message authentication failed"
	addressAlreadyUnLocked = "the address was previously unlocked"
)

var concernedErrorMap map[string]int

func init() {
	concernedErrorMap = make(map[string]int)
	concernedErrorMap[errPassword] = 4001
	concernedErrorMap[addressAlreadyUnLocked] = 4002
}

type normalError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func MakeConcernedError(err error) (errjson string, concerned bool) {
	code, ok := concernedErrorMap[err.Error()]
	if ok {
		bytes, _ := json.Marshal(&normalError{
			Code:    code,
			Message: err.Error(),
		})

		return string(bytes), ok
	} else {
		return "", false
	}
}
