package api_interface

import "encoding/json"

const (
	errPassword = "cipher: message authentication failed"
)

var concernedErrorMap map[string]int

func init() {
	concernedErrorMap = make(map[string]int)
	concernedErrorMap[errPassword] = 4001
}

type normalError struct {
	Code    int
	Message string
}

func (ne normalError) ToJson() string {
	b, e := json.Marshal(ne)
	if e != nil {
		return ""
	}
	return string(b)
}

func MakeConcernedError(err error) (errjson string, concerned bool) {
	code, ok := concernedErrorMap[err.Error()]
	if ok {
		return normalError{
			Code:    code,
			Message: err.Error(),
		}.ToJson(), ok
	} else {
		return "", false
	}
}
