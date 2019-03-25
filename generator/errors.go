package generator

import "errors"

var (
	ErrGetSnapshotOfReferredBlockFailed = errors.New("get snapshotblock of blocks referred failed")
	ErrGetFittestSnapshotBlockFailed    = errors.New("get fittest snapshotblock failed")
	ErrGetVmContextValueFailed          = errors.New("vmcontext's value is nil")
)
