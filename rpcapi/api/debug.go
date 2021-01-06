package api

type Deprecated struct {
}

func NewDeprecated() *Deprecated {
	return &Deprecated{}
}

func (p Deprecated) String() string {
	return "DeprecatedApi"
}

func (p *Deprecated) Hello() (string, error) {
	return "hello world", nil
}
