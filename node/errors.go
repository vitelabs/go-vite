package node

import "errors"

var (
	ErrDataDirUsed     = errors.New("dataDir already used by another process")
	ErrNodeStopped     = errors.New("node not started")
	ErrNodeRunning     = errors.New("node already running")
	ErrServiceUnknown  = errors.New("unknown service")
	ErrWalletConfigNil = errors.New("wallet config is nil")
	ErrViteConfigNil   = errors.New("vite config is nil")
	ErrP2PConfigNil    = errors.New("p2p config is nil")
)
