package verifier

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/vm/util"
)

var (
	ErrVerifyAccountAddrFailed = util.ErrInsufficientBalance
	ErrVerifyHashFailed        = errors.New("verify hash failed")
	ErrVerifySignatureFailed   = errors.New("verify signature failed")
	//ErrVerifyNonceFailed                   = errors.New("verify nonce failed")
	ErrVerifySnapshotOfReferredBlockFailed = errors.New("verify snapshotBlock of the referredBlock failed")
	ErrVerifyForVmGeneratorFailed          = errors.New("generator in verifier failed")
	ErrVerifyWithVmResultFailed            = errors.New("verify with vm result failed")
)
