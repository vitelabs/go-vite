package api

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
)

var preCompiledContracts = []types.Address{
	contracts.AddressMintage,
	contracts.AddressPledge,
	contracts.AddressRegister,
	contracts.AddressVote,
	contracts.AddressConsensusGroup}

type Tx struct {
	vite *vite.Vite
}

func NewTxApi(vite *vite.Vite) *Tx {
	return &Tx{
		vite: vite,
	}
}

func (t Tx) SendRawTx(block AccountBlock) error {
	log.Info("SendRawTx")
	lb, err := block.LedgerAccountBlock()
	if err != nil {
		return err
	}
	// need to remove Later
	if len(lb.Data) != 0 && !isPreCompiledContracts(lb.ToAddress) {
		return ErrorNotSupportAddNot
	}

	if len(lb.Data) != 0 && block.BlockType == ledger.BlockTypeReceive {
		return ErrorNotSupportRecvAddNote
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

func isPreCompiledContracts(address types.Address) bool {
	for _, v := range preCompiledContracts {
		if v == address {
			return true
		}
	}
	return false
}
