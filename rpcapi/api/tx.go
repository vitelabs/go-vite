package api

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
)

var preCompiledContracts = []types.Address{
	abi.AddressMintage,
	abi.AddressPledge,
	abi.AddressRegister,
	abi.AddressVote,
	abi.AddressConsensusGroup}

type Tx struct {
	vite *vite.Vite
}

func NewTxApi(vite *vite.Vite) *Tx {
	return &Tx{
		vite: vite,
	}
}

func (t Tx) SendRawTx(block *AccountBlock) error {
	log.Info("SendRawTx")
	if block == nil {
		return errors.New("empty block")
	}

	lb, err := block.LedgerAccountBlock()
	if err != nil {
		return err
	}
	// need to remove Later
	//if len(lb.Data) != 0 && !isPreCompiledContracts(lb.ToAddress) {
	//	return ErrorNotSupportAddNot
	//}
	//
	//if len(lb.Data) != 0 && block.BlockType == ledger.BlockTypeReceive {
	//	return ErrorNotSupportRecvAddNote
	//}

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

func (t Tx) SendTxWithPrivateKey(param SendTxWithPrivateKeyParam) (*AccountBlock, error) {
	amount, ok := new(big.Int).SetString(param.Amount, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: param.SelfAddr,
		ToAddress:      &param.ToAddr,
		TokenId:        &param.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Data:           param.Data,
		Difficulty:     param.Difficulty,
	}
	fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(t.vite.Chain(), nil)
	if err != nil {
		return nil, err
	}
	g, e := generator.NewGenerator(t.vite.Chain(), fitestSnapshotBlockHash, nil, &param.SelfAddr)
	if e != nil {
		return nil, e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		var privkey ed25519.PrivateKey
		privkey, e := ed25519.HexToPrivateKey(param.PrivateKey)
		if e != nil {
			return nil, nil, e
		}
		signData := ed25519.Sign(privkey, data)
		pubkey = privkey.PubByte()
		return signData, pubkey, nil
	})
	if e != nil {
		newerr, _ := TryMakeConcernedError(e)
		return nil, newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return nil, newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		if err := t.vite.Pool().AddDirectAccountBlock(param.SelfAddr, result.BlockGenList[0]); err != nil {
			return nil, err
		}
		return ledgerToRpcBlock(result.BlockGenList[0].AccountBlock, t.vite.Chain())

	} else {
		return nil, errors.New("generator gen an empty block")
	}

}

func isPreCompiledContracts(address types.Address) bool {
	for _, v := range preCompiledContracts {
		if v == address {
			return true
		}
	}
	return false
}

type SendTxWithPrivateKeyParam struct {
	SelfAddr    types.Address
	ToAddr      types.Address
	TokenTypeId types.TokenTypeId
	PrivateKey  string
	Amount      string
	Data        []byte
	Difficulty  *big.Int
}
