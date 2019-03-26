package generator

import "errors"

var (
	ErrGetSnapshotOfReferredBlockFailed = errors.New("get snapshotblock of blocks referred failed")
	ErrGetFittestSnapshotBlockFailed    = errors.New("get fittest snapshotblock failed")
	//
	ErrGetLatestAccountBlock  = errors.New("get latest account block failed")
	ErrGetLatestSnapshotBlock = errors.New("get latest snapshot block failed")
)
