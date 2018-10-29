package node

import (
	"errors"
	"syscall"
)

var (
	ErrDataDirUsed             = errors.New("dataDir already used by another process")
	ErrNodeStopped             = errors.New("node not started")
	ErrNodeRunning             = errors.New("node already running")
	ErrServiceUnknown          = errors.New("unknown service")
	ErrWalletConfigNil         = errors.New("wallet config is nil")
	ErrEntropyStorePathInvalid = errors.New("entropyStorePath is invalid")
	ErrViteConfigNil           = errors.New("vite config is nil")
	ErrP2PConfigNil            = errors.New("p2p config is nil")
	datadirInUseErrnos         = map[uint]bool{11: true, 32: true, 35: true}
)

func convertFileLockError(err error) error {
	if errno, ok := err.(syscall.Errno); ok && datadirInUseErrnos[uint(errno)] {
		return ErrDataDirUsed
	}
	return err
}
