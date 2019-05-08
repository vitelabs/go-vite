package onroad_pool

import "errors"

var (
	// OnRoadPool
	ErrOnRoadPoolNotAvailable         = errors.New("target gid's onRoadPool is not available")
	ErrCheckIsCallerFrontOnRoadFailed = errors.New("onRoadPool check the Caller's front onroad hash failed")

	//panic
	ErrLoadCallerCacheFailed = errors.New("load callerCache failed")
)
