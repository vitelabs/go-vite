package access

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type AcWriteError struct {
	Code int
	Err error
	Data interface{}
}

type ScWriteError struct {
	Code int
	Err error
	Data interface{}
}



func (scwErr ScWriteError) Error () string  {
	return scwErr.Err.Error()
}

func (acwErr AcWriteError) Error () string {
	return acwErr.Err.Error()
}

const  (
	WacDefaultErr = iota
	WacPrevHashUncorrectErr
)

const  (
	WscDefaultErr = iota
	WscNeedSyncErr
	WscPrevHashErr
)

type WscNeedSyncErrData struct {
	AccountAddress *types.Address
	TargetBlockHash *types.Hash
	TargetBlockHeight *big.Int
}
