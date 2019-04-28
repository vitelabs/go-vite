package generator

import "errors"

var (
	ErrGetLatestAccountBlock  = errors.New("get latest account block failed")
	ErrGetLatestSnapshotBlock = errors.New("get latest snapshot block failed")
)
