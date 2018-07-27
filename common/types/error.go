package types

type GetError struct {
	Code int
	Err  error
}

func (getErr GetError) Error() string {
	return getErr.Err.Error()
}
