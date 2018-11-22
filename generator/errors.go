package generator

import "github.com/pkg/errors"

var (
	ErrGetSnapshotOfReferredBlockFailed = errors.New("get snapshotblock of blocks referred failed")
	ErrGetFittestSnapshotBlockFailed    = errors.New("get fittest snapshotblock failed")
)
