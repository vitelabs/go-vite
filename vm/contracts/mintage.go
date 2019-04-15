package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"regexp"
)

type MethodMint struct{}

func (p *MethodMint) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	if block.Amount.Cmp(mintagePledgeAmount) == 0 && util.IsViteToken(block.TokenId) {
		return big.NewInt(0), nil
	} else if block.Amount.Sign() > 0 {
		return big.NewInt(0), util.ErrInvalidMethodParam
	}
	return new(big.Int).Set(mintageFee), nil
}
func (p *MethodMint) GetRefundData() []byte {
	return []byte{1}
}
func (p *MethodMint) GetSendQuota(data []byte) (uint64, error) {
	return MintGas, nil
}
func (p *MethodMint) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamMintage)
	err := abi.ABIMintage.UnpackMethod(param, abi.MethodNameMint, block.Data)
	if err != nil {
		return err
	}
	if err = CheckMintToken(*param); err != nil {
		return err
	}
	tokenId := abi.NewTokenId(block.AccountAddress, block.Height, block.PrevHash)
	block.Data, _ = abi.ABIMintage.PackMethod(
		abi.MethodNameMint,
		param.IsReIssuable,
		tokenId,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		param.MaxSupply,
		param.OwnerBurnOnly)
	return nil
}

func CheckMintToken(param abi.ParamMintage) error {
	if param.TotalSupply.Sign() <= 0 ||
		param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		param.TotalSupply.Cmp(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(param.Decimals)), nil)) < 0 ||
		len(param.TokenName) == 0 || len(param.TokenName) > tokenNameLengthMax ||
		len(param.TokenSymbol) == 0 || len(param.TokenSymbol) > tokenSymbolLengthMax {
		return util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^([a-zA-Z_]+[ ]?)*[a-zA-Z_]$", param.TokenName); !ok {
		return util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^([a-zA-Z_]+[ ]?)*[a-zA-Z_]$", param.TokenSymbol); !ok {
		return util.ErrInvalidMethodParam
	}
	if param.IsReIssuable {
		if param.MaxSupply.Cmp(param.TotalSupply) < 0 || param.MaxSupply.Cmp(helper.Tt256m1) > 0 {
			return util.ErrInvalidMethodParam
		}
	} else if param.MaxSupply.Sign() > 0 || param.OwnerBurnOnly {
		return util.ErrInvalidMethodParam
	}
	return nil
}
func (p *MethodMint) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamMintage)
	abi.ABIMintage.UnpackMethod(param, abi.MethodNameMint, sendBlock.Data)
	key := abi.GetMintageKey(param.TokenId)
	v, err := db.GetValue(key)
	util.DealWithErr(err)
	if len(v) > 0 {
		return nil, util.ErrIdCollision
	}
	var tokenInfo []byte
	if sendBlock.Amount.Sign() == 0 {
		tokenInfo, _ = abi.ABIMintage.PackVariable(
			abi.VariableNameTokenInfo,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			uint64(0),
			sendBlock.AccountAddress,
			param.IsReIssuable,
			param.MaxSupply,
			param.OwnerBurnOnly)
	} else {
		tokenInfo, _ = abi.ABIMintage.PackVariable(
			abi.VariableNameTokenInfo,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			vm.GlobalStatus().SnapshotBlock().Height+nodeConfig.params.MintPledgeHeight,
			sendBlock.AccountAddress,
			param.IsReIssuable,
			param.MaxSupply,
			param.OwnerBurnOnly)
	}
	db.SetValue(key, tokenInfo)
	ownerTokenIdListKey := abi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList, err := db.GetValue(ownerTokenIdListKey)
	util.DealWithErr(err)
	db.SetValue(ownerTokenIdListKey, abi.AppendTokenId(oldIdList, param.TokenId))

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameMint, param.TokenId))
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendReward,
			Amount:         param.TotalSupply,
			TokenId:        param.TokenId,
			Data:           []byte{},
		},
	}, nil
	return nil, nil
}

type MethodMintageCancelPledge struct{}

func (p *MethodMintageCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodMintageCancelPledge) GetRefundData() []byte {
	return []byte{2}
}

func (p *MethodMintageCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return MintageCancelPledgeGas, nil
}

func (p *MethodMintageCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	tokenId := new(types.TokenTypeId)
	if err := abi.ABIMintage.UnpackMethod(tokenId, abi.MethodNameCancelMintPledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameCancelMintPledge, *tokenId)
	return nil
}
func (p *MethodMintageCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	tokenId := new(types.TokenTypeId)
	abi.ABIMintage.UnpackMethod(tokenId, abi.MethodNameCancelMintPledge, sendBlock.Data)
	tokenInfo, err := abi.GetTokenById(db, *tokenId)
	util.DealWithErr(err)
	if tokenInfo.PledgeAddr != sendBlock.AccountAddress ||
		tokenInfo.PledgeAmount.Sign() == 0 ||
		tokenInfo.WithdrawHeight > vm.GlobalStatus().SnapshotBlock().Height {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		tokenInfo.TokenName,
		tokenInfo.TokenSymbol,
		tokenInfo.TotalSupply,
		tokenInfo.Decimals,
		tokenInfo.Owner,
		helper.Big0,
		uint64(0),
		tokenInfo.PledgeAddr,
		tokenInfo.IsReIssuable,
		tokenInfo.MaxSupply,
		tokenInfo.OwnerBurnOnly)
	db.SetValue(abi.GetMintageKey(*tokenId), newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		return []*ledger.AccountBlock{
			{
				AccountAddress: block.AccountAddress,
				ToAddress:      tokenInfo.PledgeAddr,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         tokenInfo.PledgeAmount,
				TokenId:        ledger.ViteTokenId,
				Data:           []byte{},
			},
		}, nil
	}
	return nil, nil
}

type MethodIssue struct{}

func (p *MethodIssue) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodIssue) GetRefundData() []byte {
	return []byte{4}
}
func (p *MethodIssue) GetSendQuota(data []byte) (uint64, error) {
	return IssueGas, nil
}
func (p *MethodIssue) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamIssue)
	err := abi.ABIMintage.UnpackMethod(param, abi.MethodNameIssue, block.Data)
	if err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameIssue, param.TokenId, param.Amount, param.Beneficial)
	return nil
}
func (p *MethodIssue) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamIssue)
	abi.ABIMintage.UnpackMethod(param, abi.MethodNameIssue, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenById(db, param.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress ||
		new(big.Int).Sub(oldTokenInfo.MaxSupply, oldTokenInfo.TotalSupply).Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Add(oldTokenInfo.TotalSupply, param.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.PledgeAmount,
		oldTokenInfo.WithdrawHeight,
		oldTokenInfo.PledgeAddr,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	db.SetValue(abi.GetMintageKey(param.TokenId), newTokenInfo)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameIssue, param.TokenId))
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      param.Beneficial,
			BlockType:      ledger.BlockTypeSendReward,
			Amount:         param.Amount,
			TokenId:        param.TokenId,
			Data:           []byte{},
		},
	}, nil
}

type MethodBurn struct{}

func (p *MethodBurn) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodBurn) GetRefundData() []byte {
	return []byte{5}
}
func (p *MethodBurn) GetSendQuota(data []byte) (uint64, error) {
	return BurnGas, nil
}
func (p *MethodBurn) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameBurn)
	return nil
}
func (p *MethodBurn) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	oldTokenInfo, err := abi.GetTokenById(db, sendBlock.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable ||
		(oldTokenInfo.OwnerBurnOnly && oldTokenInfo.Owner != sendBlock.AccountAddress) {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Sub(oldTokenInfo.TotalSupply, sendBlock.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.PledgeAmount,
		oldTokenInfo.WithdrawHeight,
		oldTokenInfo.PledgeAddr,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	util.SubBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	db.SetValue(abi.GetMintageKey(sendBlock.TokenId), newTokenInfo)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameBurn, sendBlock.TokenId, sendBlock.AccountAddress, sendBlock.Amount))
	return nil, nil
}

type MethodTransferOwner struct{}

func (p *MethodTransferOwner) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodTransferOwner) GetRefundData() []byte {
	return []byte{6}
}
func (p *MethodTransferOwner) GetSendQuota(data []byte) (uint64, error) {
	return TransferOwnerGas, nil
}
func (p *MethodTransferOwner) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamTransferOwner)
	err := abi.ABIMintage.UnpackMethod(param, abi.MethodNameTransferOwner, block.Data)
	if err != nil {
		return err
	}
	if param.NewOwner == block.AccountAddress {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameTransferOwner, param.TokenId, param.NewOwner)
	return nil
}
func (p *MethodTransferOwner) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTransferOwner)
	abi.ABIMintage.UnpackMethod(param, abi.MethodNameTransferOwner, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenById(db, param.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply,
		oldTokenInfo.Decimals,
		param.NewOwner,
		oldTokenInfo.PledgeAmount,
		oldTokenInfo.WithdrawHeight,
		oldTokenInfo.PledgeAddr,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	db.SetValue(abi.GetMintageKey(param.TokenId), newTokenInfo)

	oldKey := abi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList, err := db.GetValue(oldKey)
	util.DealWithErr(err)
	db.SetValue(oldKey, abi.DeleteTokenId(oldIdList, param.TokenId))
	newKey := abi.GetOwnerTokenIdListKey(param.NewOwner)
	newIdList, err := db.GetValue(newKey)
	util.DealWithErr(err)
	db.SetValue(newKey, abi.AppendTokenId(newIdList, param.TokenId))

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameTransferOwner, param.TokenId, param.NewOwner))
	return nil, nil
}

type MethodChangeTokenType struct{}

func (p *MethodChangeTokenType) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodChangeTokenType) GetRefundData() []byte {
	return []byte{7}
}
func (p *MethodChangeTokenType) GetSendQuota(data []byte) (uint64, error) {
	return ChangeTokenTypeGas, nil
}
func (p *MethodChangeTokenType) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	tokenId := new(types.TokenTypeId)
	err := abi.ABIMintage.UnpackMethod(tokenId, abi.MethodNameChangeTokenType, block.Data)
	if err != nil {
		return err
	}
	if tokenId == nil || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameChangeTokenType, &tokenId)
	return nil
}
func (p *MethodChangeTokenType) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	tokenId := new(types.TokenTypeId)
	abi.ABIMintage.UnpackMethod(tokenId, abi.MethodNameChangeTokenType, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenById(db, *tokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply,
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.PledgeAmount,
		oldTokenInfo.WithdrawHeight,
		oldTokenInfo.PledgeAddr,
		false,
		helper.Big0,
		false)
	db.SetValue(abi.GetMintageKey(*tokenId), newTokenInfo)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameChangeTokenType, *tokenId))
	return nil, nil
}
