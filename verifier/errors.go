package verifier

import "errors"

var (
	ErrVerifyHashFailed           = errors.New("verify hash failed")
	ErrVerifySignatureFailed      = errors.New("verify signature failed")
	ErrVerifyNonceFailed          = errors.New("check pow nonce failed")
	ErrVerifyForVmGeneratorFailed = errors.New("generator in verifier failed")
	ErrVerifyWithVmResultFailed   = errors.New("verify with vm result failed")

	ErrVerifyAccountTypeNotSure      = errors.New("verify accountType but is not sure")
	ErrVerifyConfirmedTimesNotEnough = errors.New("verify referred confirmedTimes not enough")
)
