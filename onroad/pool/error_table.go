package onroad_pool

import "errors"

var (
	// OnRoadPool
	ErrLoadCallerCacheFailed = errors.New("onRoadPool conflict, load callerCache failed")
	ErrBlockTypeErr          = errors.New("onRoadPool block type err")
	ErrRmTxFailed            = errors.New("onRoadPool conflict, rmTx failed")
	ErrAddTxFailed           = errors.New("onRoadPool conflict, addTx failed")
)
