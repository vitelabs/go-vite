package api

import (
	"errors"
	"github.com/vitelabs/go-vite/generator"
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

	g, err := generator.NewGenerator(t.vite.Chain(), &lb.SnapshotHash, &lb.PrevHash, &lb.AccountAddress)
	if err != nil {
		return err
	}

	result, err := g.GenerateWithBlock(lb, nil)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		return t.vite.Pool().AddDirectAccountBlock(block.AccountAddress, result.BlockGenList[0])
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}
