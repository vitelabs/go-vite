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

	ErrBalanceNotEnough = JsonRpc2Error{
		Message: util.ErrInsufficientBalance.Error(),
		Code:    -35001,
	}

	ErrQuotaNotEnough = JsonRpc2Error{
		Message: util.ErrOutOfQuota.Error(),
		Code:    -35002,
	}

	//ErrorNotSupportAddNot = JsonRpc2Error{
	//	Message: "Adding note information is not supported currently",
	//	Code:    -35003,
	//}
	//
	//ErrorNotSupportRecvAddNote = JsonRpc2Error{
	//	Message: "Adding note information in receiveBlock is not allowed",
	//	Code:    -35004,
	//}

	ErrDecryptKey = JsonRpc2Error{
		Message: walleterrors.ErrDecryptKey.Error(),
		Code:    -34001,
	}

	AddressAlreadyUnLocked = JsonRpc2Error{
		Message: walleterrors.ErrAlreadyLocked.Error(),
		Code:    -34002,
	}

	concernedErrorMap map[string]JsonRpc2Error
)

func init() {
	concernedErrorMap = make(map[string]JsonRpc2Error)
	concernedErrorMap[ErrDecryptKey.Error()] = ErrDecryptKey
	concernedErrorMap[AddressAlreadyUnLocked.Error()] = AddressAlreadyUnLocked
	concernedErrorMap[ErrBalanceNotEnough.Error()] = ErrBalanceNotEnough
	concernedErrorMap[ErrQuotaNotEnough.Error()] = ErrQuotaNotEnough
	//concernedErrorMap[ErrorNotSupportRecvAddNote.Error()] = ErrorNotSupportRecvAddNote
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
