package generator

import "errors"

var (
	// ErrGetLatestAccountBlock defines the error about failing to get latest account block from chain db
	ErrGetLatestAccountBlock = errors.New("get latest account block failed")
	// ErrGetLatestSnapshotBlock defines the error about failing to get latest snapshot block from chain db
	ErrGetLatestSnapshotBlock = errors.New("get latest snapshot block failed")

	ErrVmRunPanic = errors.New("generator_vm panic error")
)
