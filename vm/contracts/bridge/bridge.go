package bridge

import "math/big"

type Bridge interface {
	Submit(height *big.Int, content interface{}) error
	Proof(height *big.Int, content interface{}) (bool, error)
}

type InputCollector interface {
	Input(height *big.Int, content interface{}) (InputResult, error)
}

type InputResult byte

const (
	Input_Success           InputResult = 0
	Input_Failed_Error      InputResult = 1
	Input_Failed_Duplicated InputResult = 2
	Input_Failed_Not_Exist  InputResult = 3
)
