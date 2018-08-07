package api

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/ledger/errors"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
)

var (
	errBalanceNotEnough    = ledgererrors.ErrBalanceNotEnough.Error()
	errDecryptKey          = walleterrors.ErrDecryptKey.Error()
	addressAlreadyUnLocked = walleterrors.ErrAlreadyLocked.Error()
	concernedErrorMap      map[string]int
)

func init() {
	concernedErrorMap = make(map[string]int)
	concernedErrorMap[errDecryptKey] = 4001
	concernedErrorMap[addressAlreadyUnLocked] = 4002
	concernedErrorMap[errBalanceNotEnough] = 5001
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
