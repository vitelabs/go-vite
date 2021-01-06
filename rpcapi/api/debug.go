package api

type Debug struct {
}

func NewDebug() *Debug {
	return &Debug{}
}

func (p Debug) String() string {
	return "DebugApi"
}

func (p *Debug) Hello() (string, error) {
	return "hello world", nil
}
