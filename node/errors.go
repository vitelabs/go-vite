package node

import "errors"

var (
	ErrDataDirUsed    = errors.New("dataDir already used by another process")
	ErrNodeStopped    = errors.New("node not started")
	ErrNodeRunning    = errors.New("node already running")
	ErrServiceUnknown = errors.New("unknown service")
)
