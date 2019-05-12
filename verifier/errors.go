package verifier

import "errors"

var (
	ErrVerifyForVmGeneratorFailed = errors.New("generator in verifier failed")

	//ErrVerifyAccountAddrFailed       = errors.New("account address doesn't exist, need receiveTx for more balance first")
	ErrVerifyAccountTypeNotSure      = errors.New("general account's sendBlock.Height must be larger than 1")
	ErrVerifyConfirmedTimesNotEnough = errors.New("verify referred confirmedTimes not enough")

	ErrVerifyHashFailed                    = errors.New("verify hash failed")
	ErrVerifySignatureFailed               = errors.New("verify signature failed")
	ErrVerifyNonceFailed                   = errors.New("check pow nonce failed")
	ErrVerifySnapshotOfReferredBlockFailed = errors.New("verify snapshotBlock of the referredBlock failed")
	ErrVerifyPrevBlockFailed               = errors.New("verify prevBlock failed, incorrect use of prevHash or fork happened")
	ErrVerifyRPCBlockPendingState          = errors.New("verify referred block failed, pending for something")
)
