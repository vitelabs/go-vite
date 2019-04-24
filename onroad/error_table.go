package onroad

import "github.com/go-errors/errors"

var (
	//manager
	ErrNotSyncDone = errors.New("network synchronization is not complete")

	// OnRoadPool
	ErrRmTxFailed            = errors.New("onRoadPool conflict, rmTx failed")
	ErrAddTxFailed           = errors.New("onRoadPool conflict, addTx failed")
	ErrLoadCallerCacheFailed = errors.New("onRoadPool conflict, load callerCache failed")
	ErrBlockTypeErr          = errors.New("onRoadPool block type err")
)
