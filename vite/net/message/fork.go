package message

type Fork struct {
	List []*BlockID
}

func (f *Fork) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *Fork) Deserialize(buf []byte) error {
	panic("implement me")
}
