package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"regexp"
)

type MethodMintage struct{}

func (p *MethodMintage) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	if block.Amount.Cmp(mintagePledgeAmount) == 0 && util.IsViteToken(block.TokenId) {
		// Pledge ViteToken to mintage
		return big.NewInt(0), nil
	} else if block.Amount.Sign() > 0 {
		return big.NewInt(0), errors.New("invalid amount")
	}
	// Destroy ViteToken to mintage
	return new(big.Int).Set(mintageFee), nil
}

func (p *MethodMintage) GetRefundData() []byte {
	return []byte{1}
}
func (p *MethodMintage) GetQuota() uint64 {
	return MintageGas
}

func (p *MethodMintage) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	param := new(cabi.ParamMintage)
	err = cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameMintage, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = CheckToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := cabi.NewTokenId(block.AccountAddress, block.Height, block.PrevHash, block.SnapshotHash)
	if cabi.GetTokenById(db, tokenId) != nil {
		return quotaLeft, util.ErrIdCollision
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(
		cabi.MethodNameMintage,
		tokenId,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals)
	return quotaLeft, nil
}
func CheckToken(param cabi.ParamMintage) error {
	if param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		param.TotalSupply.Cmp(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(param.Decimals)), nil)) < 0 ||
		len(param.TokenName) == 0 || len(param.TokenName) > tokenNameLengthMax ||
		len(param.TokenSymbol) == 0 || len(param.TokenSymbol) > tokenSymbolLengthMax {
		return errors.New("invalid token param")
	}
	if ok, _ := regexp.MatchString("^([a-zA-Z_]+[ ]?)*[a-zA-Z_]$", param.TokenName); !ok {
		return errors.New("invalid token name")
	}
	if ok, _ := regexp.MatchString("^([a-zA-Z_]+[ ]?)*[a-zA-Z_]$", param.TokenSymbol); !ok {
		return errors.New("invalid token symbol")
	}
	return nil
}
func (p *MethodMintage) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamMintage)
	cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameMintage, sendBlock.Data)
	key := cabi.GetMintageKey(param.TokenId)
	if len(db.GetStorage(&block.AccountAddress, key)) > 0 {
		return nil, util.ErrIdCollision
	}
	var tokenInfo []byte
	if sendBlock.Amount.Sign() == 0 {
		tokenInfo, _ = cabi.ABIMintage.PackVariable(
			cabi.VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			uint64(0))
	} else {
		tokenInfo, _ = cabi.ABIMintage.PackVariable(
			cabi.VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			db.CurrentSnapshotBlock().Height+nodeConfig.params.MintagePledgeHeight)
	}
	db.SetStorage(key, tokenInfo)
	return []*SendBlock{
		{
			block,
			sendBlock.AccountAddress,
			ledger.BlockTypeSendReward,
			param.TotalSupply,
			param.TokenId,
			[]byte{},
		},
	}, nil
}

type MethodMintageCancelPledge struct{}

func (p *MethodMintageCancelPledge) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodMintageCancelPledge) GetRefundData() []byte {
	return []byte{2}
}
func (p *MethodMintageCancelPledge) GetQuota() uint64 {
	return MintageCancelPledgeGas
}

func (p *MethodMintageCancelPledge) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() > 0 {
		return quotaLeft, errors.New("invalid block data")
	}
	tokenId := new(types.TokenTypeId)
	if err = cabi.ABIMintage.UnpackMethod(tokenId, cabi.MethodNameMintageCancelPledge, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	return quotaLeft, nil
}
func (p *MethodMintageCancelPledge) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	tokenId := new(types.TokenTypeId)
	cabi.ABIMintage.UnpackMethod(tokenId, cabi.MethodNameMintageCancelPledge, sendBlock.Data)
	storageKey := cabi.GetMintageKey(*tokenId)
	tokenInfo := new(types.TokenInfo)
	cabi.ABIMintage.UnpackVariable(tokenInfo, cabi.VariableNameMintage, db.GetStorage(&block.AccountAddress, storageKey))

	if tokenInfo.Owner != sendBlock.AccountAddress ||
		tokenInfo.PledgeAmount.Sign() == 0 ||
		tokenInfo.WithdrawHeight > db.CurrentSnapshotBlock().Height {
		return nil, errors.New("cannot withdraw mintage pledge, status error")
	}

	newTokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameMintage,
		tokenInfo.TokenName,
		tokenInfo.TokenSymbol,
		tokenInfo.TotalSupply,
		tokenInfo.Decimals,
		tokenInfo.Owner,
		big.NewInt(0),
		uint64(0))
	db.SetStorage(storageKey, newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		return []*SendBlock{
			{
				block,
				tokenInfo.Owner,
				ledger.BlockTypeSendCall,
				tokenInfo.PledgeAmount,
				ledger.ViteTokenId,
				[]byte{},
			},
		}, nil
	}
	return nil, nil
}
