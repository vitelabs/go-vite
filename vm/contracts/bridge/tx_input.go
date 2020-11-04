package bridge

import (
	"math/big"
	"math/rand"
)

type inputSimple struct {
	ok map[int]bool
	bg Bridge
}

type inputSimpleTx struct {
	txId int
}

func newInputSimpleTx(bg Bridge) (InputCollector, error) {
	return &inputSimple{
		ok: make(map[int]bool),
		bg: bg,
	}, nil
}

func randSimpleTx() *inputSimpleTx {
	return &inputSimpleTx{txId: rand.Int()}
}

func (input *inputSimple) Input(height *big.Int, content interface{}) (InputResult, error) {
	tx := content.(*inputSimpleTx)
	if input.ok[tx.txId] {
		return Input_Failed_Duplicated, nil
	}
	proof, err := input.bg.Proof(height, content)
	if err != nil {
		return Input_Failed_Error, err
	}
	if proof {
		input.ok[tx.txId] = true
		return Input_Success, nil
	}
	return Input_Failed_Error, nil
}
