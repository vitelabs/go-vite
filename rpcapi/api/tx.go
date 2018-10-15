package api

import (
	"errors"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite"
)

type Tx struct {
	vite *vite.Vite
}

func NewTxApi(vite *vite.Vite) *Tx {
	return &Tx{
		vite: vite,
	}
}

func (t Tx) SendRawTx(block AccountBlock) error {
	lb, err := block.LedgerAccountBlock()
	if err != nil {
		return err
	}

	v := verifier.NewAccountVerifier(t.vite.Chain(), t.vite.Consensus())

	blocks, err := v.VerifyforRPC(lb)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return newerr
	}

	if len(blocks) > 0 && blocks[0] != nil {
		return t.vite.Pool().AddDirectAccountBlock(block.AccountAddress, blocks[0])
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}
