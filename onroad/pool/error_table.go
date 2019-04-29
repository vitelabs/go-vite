package onroad_pool

import "errors"

var (
	// OnRoadPool
	ErrOnRoadPoolNotAvailable = errors.New("target gid's onRoadPool is not available")

	ErrBlockTypeErr = errors.New("onRoadPool block type err")
	//panic
	ErrLoadCallerCacheFailed = errors.New("onRoadPool conflict, load callerCache failed")
	ErrRmTxFailed            = errors.New("onRoadPool conflict, rmTx failed")
	ErrAddTxFailed           = errors.New("onRoadPool conflict, addTx failed")
)
