package bridge

import "math/big"

type bridgeSimple struct {
	m map[string]bool
}

func newBridgeSimple() (Bridge, error) {
	b := &bridgeSimple{}
	b.m = make(map[string]bool)
	return b, nil
}

func (b *bridgeSimple) Submit(height *big.Int, content interface{}) error {
	b.m[height.String()] = true
	return nil
}

func (b *bridgeSimple) Proof(height *big.Int, content interface{}) (bool, error) {
	return b.m[height.String()], nil
}
