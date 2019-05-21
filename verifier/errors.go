package verifier

import "errors"

// Errors that the external module needs to be aware of.
var (
	ErrVerifyForVMGeneratorFailed = errors.New("generator in verifier failed")

	ErrVerifyAccountTypeNotSure      = errors.New("general account's sendBlock.Height must be larger than 1")
	ErrVerifyConfirmedTimesNotEnough = errors.New("verify referred confirmedTimes not enough")

	ErrVerifyHashFailed           = errors.New("verify hash failed")
	ErrVerifySignatureFailed      = errors.New("verify signature failed")
	ErrVerifyNonceFailed          = errors.New("check pow nonce failed")
	ErrVerifyPrevBlockFailed      = errors.New("verify prevBlock failed, incorrect use of prevHash or fork happened")
	ErrVerifyRPCBlockPendingState = errors.New("verify referred block failed, pending for them")
)
