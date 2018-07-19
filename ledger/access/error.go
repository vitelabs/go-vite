package access


type AcWriteError struct {
	Code int
	Err error
	Data interface{}
}

func (werr AcWriteError) Error () string {
	return werr.Err.Error()
}


const  (
	WacDefaultErr = iota
	WacPrevHashUncorrectErr
)
