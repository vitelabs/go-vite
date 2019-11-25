package verifier

import "errors"

// Errors that the external module needs to be aware of.
var (
	// can retry
	ErrVerifyVmGeneratorFailed           = errors.New("generator in verifier run failed")
	ErrVerifyAccountNotInvalid           = errors.New("general account's sendBlock.Height must be larger than 1")
	ErrVerifyContractMetaNotExists       = errors.New("contract meta not exists")
	ErrVerifyConfirmedTimesNotEnough     = errors.New("verify referred send's confirmedTimes not enough")
	ErrVerifySeedConfirmedTimesNotEnough = errors.New("verify referred send's seed confirmedTimes not enough")

	ErrVerifyHashFailed           = errors.New("verify hash failed")
	ErrVerifySignatureFailed      = errors.New("verify signature failed")
	ErrVerifyNonceFailed          = errors.New("check pow nonce failed")
	ErrVerifyPrevBlockFailed      = errors.New("verify prevBlock failed, incorrect use of prevHash or fork happened")
	ErrVerifyRPCBlockPendingState = errors.New("verify referred block failed, pending for them")

	// check data, can't retry
	ErrVerifyDependentSendBlockNotExists   = errors.New("receive's dependent send block is not exists on chain")
	ErrVerifyPowNotEligible                = errors.New("verify that it's not eligible to do pow")
	ErrVerifyProducerIllegal               = errors.New("verify that the producer is illegal")
	ErrVerifyBlockFieldData                = errors.New("verify that block field data is illegal")
	ErrVerifyContractReceiveSequenceFailed = errors.New("verify that contract's receive sequence is illegal")
	ErrVerifySendIsAlreadyReceived         = errors.New("block is already received successfully")
	ErrVerifyVmResultInconsistent          = errors.New("inconsistent execution results in vm")
)

type VerifierError struct {
	err    string
	detail *string
}

func (e VerifierError) Error() string {
	return e.err
}

func (e VerifierError) Detail() string {
	if e.detail == nil {
		return ""
	}
	return *e.detail
}

func newError(err string) *VerifierError {
	return &VerifierError{err: err}
}

func newDetailError(err, detail string) *VerifierError {
	return &VerifierError{err: err, detail: &detail}
}
