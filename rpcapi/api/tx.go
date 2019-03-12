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
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

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
	//if len(lb.Data) != 0 && !isBuiltinContracts(lb.ToAddress) {
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

	if param.Amount == nil {
		return nil, errors.New("amount is nil")
	}

	if param.SelfAddr == nil {
		return nil, errors.New("selfAddr is nil")
	}

	if param.ToAddr == nil && param.BlockType != ledger.BlockTypeSendCreate {
		return nil, errors.New("toAddr is nil")
	}

	if param.PrivateKey == nil {
		return nil, errors.New("privateKey is nil")
	}

	var d *big.Int = nil
	if param.Difficulty != nil {
		t, ok := new(big.Int).SetString(*param.Difficulty, 10)
		if !ok {
			return nil, ErrStrToBigInt
		}
		d = t
	}

	amount, ok := new(big.Int).SetString(*param.Amount, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}
	var blockType byte
	if param.BlockType > 0 {
		blockType = param.BlockType
	} else {
		blockType = ledger.BlockTypeSendCall
	}
	msg := &generator.IncomingMessage{
		BlockType:      blockType,
		AccountAddress: *param.SelfAddr,
		ToAddress:      param.ToAddr,
		TokenId:        &param.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Data:           param.Data,
		Difficulty:     d,
	}
	_, fitestSnapshotBlockHash, err := generator.GetFittestGeneratorSnapshotHash(t.vite.Chain(), &msg.AccountAddress, nil, false)
	if err != nil {
		return nil, err
	}
	g, e := generator.NewGenerator(t.vite.Chain(), fitestSnapshotBlockHash, param.PreBlockHash, param.SelfAddr)
	if e != nil {
		return nil, e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		var privkey ed25519.PrivateKey
		privkey, e := ed25519.HexToPrivateKey(*param.PrivateKey)
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
		if err := t.vite.Pool().AddDirectAccountBlock(*param.SelfAddr, result.BlockGenList[0]); err != nil {
			return nil, err
		}
		return ledgerToRpcBlock(result.BlockGenList[0].AccountBlock, t.vite.Chain())

	} else {
		return nil, errors.New("generator gen an empty block")
	}

}

type SendTxWithPrivateKeyParam struct {
	SelfAddr     *types.Address    `json:"selfAddr"`
	ToAddr       *types.Address    `json:"toAddr"`
	TokenTypeId  types.TokenTypeId `json:"tokenTypeId"`
	PrivateKey   *string           `json:"privateKey"` //hex16
	Amount       *string           `json:"amount"`
	Data         []byte            `json:"data"` //base64
	Difficulty   *string           `json:"difficulty,omitempty"`
	PreBlockHash *types.Hash       `json:"preBlockHash,omitempty"`
	BlockType    byte              `json:"blockType"`
}

type CalcPoWDifficultyParam struct {
	SelfAddr     types.Address `json:"selfAddr"`
	PrevHash     types.Hash    `json:"prevHash"`
	SnapshotHash types.Hash    `json:"snapshotHash"`

	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddr"`
	Data      []byte         `json:"data"`

	UsePledgeQuota bool `json:"usePledgeQuota"`
}

type CalcPoWDifficultyResult struct {
	quotaRequired uint64 `json:"quota"`
	difficulty    string `json:"difficulty"`
}

func (t Tx) CalcPoWDifficulty(param CalcPoWDifficultyParam) (result *CalcPoWDifficultyResult, err error) {
	block := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		AccountAddress: param.SelfAddr,
		PrevHash:       param.PrevHash,
		SnapshotHash:   param.SnapshotHash,
		Data:           param.Data,
	}
	if param.ToAddr != nil {
		block.ToAddress = *param.ToAddr
	} else if param.BlockType == ledger.BlockTypeSendCall {
		return nil, errors.New("toAddr is nil")
	}
	quotaRequired, err := quota.GetQuotaRequired(block)
	if err != nil {
		return nil, err
	}

	db, err := vm_context.NewVmContext(t.vite.Chain(), &param.SnapshotHash, &param.PrevHash, &param.SelfAddr)
	if err != nil {
		return nil, err
	}
	pledgeAmount := abi.GetPledgeBeneficialAmount(db, param.SelfAddr)
	q, err := quota.GetPledgeQuota(db, param.SelfAddr, pledgeAmount)
	if param.UsePledgeQuota {
		if err != nil {
			return nil, err
		}
		if q.Current() >= quotaRequired {
			return &CalcPoWDifficultyResult{quotaRequired, ""}, nil
		}
	}

	if !quota.CanPoW(db, param.SelfAddr) {
		return nil, util.ErrCalcPoWTwice
	}
	d, err := quota.CalcPoWDifficulty(quotaRequired, q, pledgeAmount)
	if err != nil {
		return nil, err
	}
	return &CalcPoWDifficultyResult{quotaRequired, d.String()}, nil
}

func isPoW(nonce []byte) bool {
	return len(nonce) > 0
}
