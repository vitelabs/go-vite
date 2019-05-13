package api

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/verifier"
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
	IllegalNodeTime = errors.New("The node time is inaccurate, quite different from the time of latest snapshot block.")

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

	ErrVmMethodNotFound = JsonRpc2Error{
		Message: util.ErrAbiMethodNotFound.Error(),
		Code:    -35006,
	}

	ErrVmInvalidConfirmTime = JsonRpc2Error{
		Message: util.ErrInvalidConfirmTime.Error(),
		Code:    -35007,
	}

	ErrVmContractNotExists = JsonRpc2Error{
		Message: util.ErrContractNotExists.Error(),
		Code:    -35008,
	}

	ErrVmNoReliableStatus = JsonRpc2Error{
		Message: util.ErrNoReliableStatus.Error(),
		Code:    -35009,
	}

	ErrVmInvalidQuotaRatio = JsonRpc2Error{
		Message: util.ErrInvalidQuotaRatio.Error(),
		Code:    -35010,
	}

	// -36001 ~ -36999 verifier_account
	ErrVerifyAccountAddr = JsonRpc2Error{
		Message: verifier.ErrVerifyAccountTypeNotSure.Error(),
		Code:    -36001,
	}
	ErrVerifyHash = JsonRpc2Error{
		Message: verifier.ErrVerifyHashFailed.Error(),
		Code:    -36002,
	}
	ErrVerifySignature = JsonRpc2Error{
		Message: verifier.ErrVerifySignatureFailed.Error(),
		Code:    -36003,
	}
	ErrVerifyNonce = JsonRpc2Error{
		Message: verifier.ErrVerifyNonceFailed.Error(),
		Code:    -36004,
	}
	ErrVerifyPrevBlock = JsonRpc2Error{
		Message: verifier.ErrVerifyPrevBlockFailed.Error(),
		Code:    -36005,
	}
	ErrVerifyRPCBlockIsPending = JsonRpc2Error{
		Message: verifier.ErrVerifyRPCBlockPendingState.Error(),
		Code:    -36006,
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
	concernedErrorMap[ErrVmMethodNotFound.Error()] = ErrVmMethodNotFound
	concernedErrorMap[ErrVmInvalidConfirmTime.Error()] = ErrVmInvalidConfirmTime
	concernedErrorMap[ErrVmContractNotExists.Error()] = ErrVmContractNotExists
	concernedErrorMap[ErrVmNoReliableStatus.Error()] = ErrVmNoReliableStatus
	concernedErrorMap[ErrVmInvalidQuotaRatio.Error()] = ErrVmInvalidQuotaRatio

	concernedErrorMap[ErrVerifyAccountAddr.Error()] = ErrVerifyAccountAddr
	concernedErrorMap[ErrVerifyHash.Error()] = ErrVerifyHash
	concernedErrorMap[ErrVerifySignature.Error()] = ErrVerifySignature
	concernedErrorMap[ErrVerifyNonce.Error()] = ErrVerifyNonce
	concernedErrorMap[ErrVerifyPrevBlock.Error()] = ErrVerifyPrevBlock
	concernedErrorMap[ErrVerifyRPCBlockIsPending.Error()] = ErrVerifyRPCBlockIsPending
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
