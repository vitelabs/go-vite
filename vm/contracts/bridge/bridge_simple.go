package bridge

type bridgeSimple struct {
	m map[uint64]bool
}

func (b *bridgeSimple) Init(content interface{}) error {
	return nil
}

type simpleHeader struct {
	height uint64
}

func newBridgeSimple() (Bridge, error) {
	b := &bridgeSimple{}
	b.m = make(map[uint64]bool)
	return b, nil
}

func (b *bridgeSimple) Submit(content interface{}) error {
	header := content.(*simpleHeader)
	b.m[header.height] = true
	return nil
}

func (b bridgeSimple) Proof(content interface{}) (bool, error) {
	header := content.(*simpleHeader)
	return b.m[header.height], nil
}
