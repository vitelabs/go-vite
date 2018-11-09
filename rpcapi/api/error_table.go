package api

import (
	"github.com/vitelabs/go-vite/vm/util"
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
	// ErrNotSupport = errors.New("not support this method")

	ErrDecryptKey = JsonRpc2Error{
		Message: walleterrors.ErrDecryptEntropy.Error(),
		Code:    -34001,
	}

	// -35001 ~ -35999 vm execution error
	ErrBalanceNotEnough = JsonRpc2Error{
		Message: util.ErrInsufficientBalance.Error(),
		Code:    -35001,
	}

	ErrQuotaNotEnough = JsonRpc2Error{
		Message: util.ErrOutOfQuota.Error(),
		Code:    -35002,
	}

	ErrVmIdCollision = JsonRpc2Error{
		Message: util.ErrIdCollision.Error(),
		Code:    -35003,
	}
	ErrVmInvaildBlockData = JsonRpc2Error{
		Message: util.ErrInvalidMethodParam.Error(),
		Code:    -35004,
	}
	ErrVmCalPoWTwice = JsonRpc2Error{
		Message: util.ErrCalcPoWTwice.Error(),
		Code:    -35005,
	}

	concernedErrorMap map[string]JsonRpc2Error
)

func init() {
	concernedErrorMap = make(map[string]JsonRpc2Error)
	concernedErrorMap[ErrDecryptKey.Error()] = ErrDecryptKey

	concernedErrorMap[ErrBalanceNotEnough.Error()] = ErrBalanceNotEnough
	concernedErrorMap[ErrQuotaNotEnough.Error()] = ErrQuotaNotEnough

	concernedErrorMap[ErrVmIdCollision.Error()] = ErrVmIdCollision
	concernedErrorMap[ErrVmInvaildBlockData.Error()] = ErrVmInvaildBlockData
	concernedErrorMap[ErrVmCalPoWTwice.Error()] = ErrVmCalPoWTwice
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
