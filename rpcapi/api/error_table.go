package api

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
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
		Message: util.ErrIDCollision.Error(),
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

	ErrVmInvalidResponseLatency = JsonRpc2Error{
		Message: util.ErrInvalidResponseLatency.Error(),
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

	ErrVmInvalidQuotaMultiplier = JsonRpc2Error{
		Message: util.ErrInvalidQuotaMultiplier.Error(),
		Code:    -35010,
	}
	ErrVmPoWNotSupported = JsonRpc2Error{
		Message: ErrPoWNotSupportedUnderCongestion.Error(),
		Code:    -35011,
	}
	ErrVmQuotaLimitReached = JsonRpc2Error{
		Message: util.ErrBlockQuotaLimitReached.Error(),
		Code:    -35012,
	}
	ErrVmInvalidRandomDegree = JsonRpc2Error{
		Message: util.ErrInvalidRandomDegree.Error(),
		Code:    -35013,
	}

	// -36001 ~ -36999 verifier_account
	ErrVerifyAccountAddr = JsonRpc2Error{
		Message: verifier.ErrVerifyAccountNotInvalid.Error(),
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
	ErrVerifyDependentSendBlockNotExists = JsonRpc2Error{
		Message: verifier.ErrVerifyDependentSendBlockNotExists.Error(),
		Code:    -36007,
	}
	ErrVerifyPowQualificationNotEnough = JsonRpc2Error{
		Message: verifier.ErrVerifyPowNotEligible.Error(),
		Code:    -36008,
	}
	ErrVerifyProducerIllegal = JsonRpc2Error{
		Message: verifier.ErrVerifyProducerIllegal.Error(),
		Code:    -36009,
	}
	ErrVerifyBlockFieldData = JsonRpc2Error{
		Message: verifier.ErrVerifyBlockFieldData.Error(),
		Code:    -36010,
	}
	ErrVerifyIsAlreadyReceived = JsonRpc2Error{
		Message: verifier.ErrVerifySendIsAlreadyReceived.Error(),
		Code:    -36011,
	}
	ErrVerifyVmResultInconsistent = JsonRpc2Error{
		Message: verifier.ErrVerifyVmResultInconsistent.Error(),
		Code:    -36012,
	}

	// -37001 ~ -37999 contracts_dex
	ErrComposeOrderIdFail = JsonRpc2Error{
		Message: dex.ComposeOrderIdFailErr.Error(),
		Code:    -37001,
	}
	ErrDexInvalidOrderType = JsonRpc2Error{
		Message: dex.InvalidOrderTypeErr.Error(),
		Code:    -37002,
	}
	ErrDexInvalidOrderPrice = JsonRpc2Error{
		Message: dex.InvalidOrderPriceErr.Error(),
		Code:    -37003,
	}
	ErrDexInvalidOrderQuantity = JsonRpc2Error{
		Message: dex.InvalidOrderQuantityErr.Error(),
		Code:    -37004,
	}
	ErrDexOrderAmountTooSmall = JsonRpc2Error{
		Message: dex.OrderAmountTooSmallErr.Error(),
		Code:    -37005,
	}
	ErrDexTradeMarketExists = JsonRpc2Error{
		Message: dex.TradeMarketExistsErr.Error(),
		Code:    -37006,
	}
	ErrDexTradeMarketNotExists = JsonRpc2Error{
		Message: dex.TradeMarketNotExistsErr.Error(),
		Code:    -37007,
	}
	ErrDexTradeOrderNotExistsErr = JsonRpc2Error{
		Message: dex.OrderNotExistsErr.Error(),
		Code:    -37008,
	}
	ErrDexCancelOrderOwnerInvalid = JsonRpc2Error{
		Message: dex.CancelOrderOwnerInvalidErr.Error(),
		Code:    -37009,
	}
	ErrDexCancelOrderInvalidStatus = JsonRpc2Error{
		Message: dex.CancelOrderInvalidStatusErr.Error(),
		Code:    -37010,
	}
	ErrDexTradeMarketInvalidQuoteToken = JsonRpc2Error{
		Message: dex.TradeMarketInvalidQuoteTokenErr.Error(),
		Code:    -37011,
	}
	ErrDexTradeMarketInvalidTokenPair = JsonRpc2Error{
		Message: dex.TradeMarketInvalidTokenPairErr.Error(),
		Code:    -37012,
	}

	ErrDexFundUserNotExists = JsonRpc2Error{
		Message: dex.DexFundUserNotExists.Error(),
		Code:    -37013,
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
	concernedErrorMap[ErrVmInvalidResponseLatency.Error()] = ErrVmInvalidResponseLatency
	concernedErrorMap[ErrVmContractNotExists.Error()] = ErrVmContractNotExists
	concernedErrorMap[ErrVmNoReliableStatus.Error()] = ErrVmNoReliableStatus
	concernedErrorMap[ErrVmInvalidQuotaMultiplier.Error()] = ErrVmInvalidQuotaMultiplier
	concernedErrorMap[ErrVmPoWNotSupported.Error()] = ErrVmPoWNotSupported
	concernedErrorMap[ErrVmQuotaLimitReached.Error()] = ErrVmQuotaLimitReached
	concernedErrorMap[ErrVmInvalidRandomDegree.Error()] = ErrVmInvalidRandomDegree

	concernedErrorMap[ErrVerifyAccountAddr.Error()] = ErrVerifyAccountAddr
	concernedErrorMap[ErrVerifyHash.Error()] = ErrVerifyHash
	concernedErrorMap[ErrVerifySignature.Error()] = ErrVerifySignature
	concernedErrorMap[ErrVerifyNonce.Error()] = ErrVerifyNonce
	concernedErrorMap[ErrVerifyPrevBlock.Error()] = ErrVerifyPrevBlock
	concernedErrorMap[ErrVerifyRPCBlockIsPending.Error()] = ErrVerifyRPCBlockIsPending
	concernedErrorMap[ErrVerifyDependentSendBlockNotExists.Error()] = ErrVerifyDependentSendBlockNotExists
	concernedErrorMap[ErrVerifyPowQualificationNotEnough.Error()] = ErrVerifyPowQualificationNotEnough
	concernedErrorMap[ErrVerifyProducerIllegal.Error()] = ErrVerifyProducerIllegal
	concernedErrorMap[ErrVerifyBlockFieldData.Error()] = ErrVerifyBlockFieldData
	concernedErrorMap[ErrVerifyIsAlreadyReceived.Error()] = ErrVerifyIsAlreadyReceived
	concernedErrorMap[ErrVerifyVmResultInconsistent.Error()] = ErrVerifyVmResultInconsistent

	concernedErrorMap[ErrComposeOrderIdFail.Error()] = ErrComposeOrderIdFail
	concernedErrorMap[ErrDexInvalidOrderType.Error()] = ErrDexInvalidOrderType
	concernedErrorMap[ErrDexInvalidOrderPrice.Error()] = ErrDexInvalidOrderPrice
	concernedErrorMap[ErrDexInvalidOrderQuantity.Error()] = ErrDexInvalidOrderQuantity
	concernedErrorMap[ErrDexOrderAmountTooSmall.Error()] = ErrDexOrderAmountTooSmall
	concernedErrorMap[ErrDexTradeMarketExists.Error()] = ErrDexTradeMarketExists
	concernedErrorMap[ErrDexTradeMarketNotExists.Error()] = ErrDexTradeMarketNotExists
	concernedErrorMap[ErrDexTradeOrderNotExistsErr.Error()] = ErrDexTradeOrderNotExistsErr
	concernedErrorMap[ErrDexCancelOrderOwnerInvalid.Error()] = ErrDexCancelOrderOwnerInvalid
	concernedErrorMap[ErrDexCancelOrderInvalidStatus.Error()] = ErrDexCancelOrderInvalidStatus
	concernedErrorMap[ErrDexTradeMarketInvalidQuoteToken.Error()] = ErrDexTradeMarketInvalidQuoteToken
	concernedErrorMap[ErrDexTradeMarketInvalidTokenPair.Error()] = ErrDexTradeMarketInvalidTokenPair
	concernedErrorMap[ErrDexFundUserNotExists.Error()] = ErrDexFundUserNotExists

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
