package pmverifier

import "errors"

var (
	ErrVerifyAccountAddrFailed             = errors.New("account address doesn't exist, need receiveTx for more balance first")
	ErrVerifyHashFailed                    = errors.New("verify hash failed")
	ErrVerifySignatureFailed               = errors.New("verify signature failed")
	ErrVerifyNonceFailed                   = errors.New("check pow nonce failed")
	ErrVerifySnapshotOfReferredBlockFailed = errors.New("verify snapshotBlock of the referredBlock failed")
	ErrVerifyForVmGeneratorFailed          = errors.New("generator in verifier failed")
	ErrVerifyWithVmResultFailed            = errors.New("verify with vm result failed")

	ErrVerifyAccountTypeNotSure      = errors.New("verify accountType but is not sure")
	ErrVerifyConfirmedTimesNotEnough = errors.New("verify referred confirmedTimes not enough")
)
