package verifier

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/vm/util"
)

var (
	ErrVerifyAccountAddrFailed             = util.ErrInsufficientBalance
	ErrVerifyHashFailed                    = errors.New("verify hash failed")
	ErrVerifySignatureFailed               = errors.New("verify signature failed")
	ErrVerifySnapshotOfReferredBlockFailed = errors.New("verify snapshotBlock of the referredBlock failed")
)
