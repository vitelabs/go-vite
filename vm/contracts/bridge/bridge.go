package bridge

type Bridge interface {
	Init(content interface{}) error
	Submit(content interface{}) error
	Proof(content interface{}) (bool, error)
}

type InputCollector interface {
	Input(content interface{}) (InputResult, error)
}

type InputResult byte

const (
	Input_Success           InputResult = 0
	Input_Failed_Error      InputResult = 1
	Input_Failed_Duplicated InputResult = 2
	Input_Failed_Not_Exist  InputResult = 3
)
