package api

import (
	"github.com/vitelabs/go-vite/ledger/errors"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
)

type JsonRpc2Error struct {
	Message string
	Code    int
}

func (e JsonRpc2Error) Error() string {
	return e.Message
}

func (e JsonRpc2Error) ErrorCode() int {
	return e.Code
}

var (
	errBalanceNotEnough = JsonRpc2Error{
		Message: ledgererrors.ErrBalanceNotEnough.Error(),
		Code:    5001,
	}

	errDecryptKey = JsonRpc2Error{
		Message: walleterrors.ErrDecryptKey.Error(),
		Code:    4001,
	}

	addressAlreadyUnLocked = JsonRpc2Error{
		Message: walleterrors.ErrAlreadyLocked.Error(),
		Code:    4002,
	}

	concernedErrorMap map[string]JsonRpc2Error
)

func init() {
	concernedErrorMap = make(map[string]JsonRpc2Error)
	concernedErrorMap[errDecryptKey.Error()] = errDecryptKey
	concernedErrorMap[addressAlreadyUnLocked.Error()] = addressAlreadyUnLocked
	concernedErrorMap[errBalanceNotEnough.Error()] = errBalanceNotEnough
}

func TryMakeConcernedError(err error) (newerr error, concerned bool) {
	if err == nil {
		return nil, false
	}
	rerr, ok := concernedErrorMap[err.Error()]
	if ok {
		return rerr, ok
	}
	return err, false

}
