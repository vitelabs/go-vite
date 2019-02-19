package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/fork"
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
	if fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
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
	if param.TotalSupply.Sign() <= 0 ||
		param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
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
	block.Data, _ = cabi.ABIMintage.PackMethod(cabi.MethodNameMintageCancelPledge, *tokenId)
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

type MethodMint struct{}

func (p *MethodMint) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	if block.Amount.Sign() > 0 {
		return big.NewInt(0), errors.New("invalid amount")
	}
	return new(big.Int).Set(mintageFee), nil
}
func (p *MethodMint) GetRefundData() []byte {
	return []byte{3}
}
func (p *MethodMint) GetQuota() uint64 {
	return MintGas
}
func (p *MethodMint) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if !fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	param := new(cabi.ParamMintage)
	err = cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameMint, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = CheckMintToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := cabi.NewTokenId(block.AccountAddress, block.Height, block.PrevHash, block.SnapshotHash)
	if cabi.GetTokenById(db, tokenId) != nil {
		return quotaLeft, util.ErrIdCollision
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(
		cabi.MethodNameMint,
		param.IsReIssuable,
		tokenId,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		param.MaxSupply,
		param.OwnerBurnOnly)
	return quotaLeft, nil
}
func CheckMintToken(param cabi.ParamMintage) error {
	if err := CheckToken(param); err != nil {
		return err
	}
	if param.IsReIssuable {
		if param.MaxSupply.Cmp(param.TotalSupply) < 0 || param.MaxSupply.Cmp(helper.Tt256m1) > 0 {
			return errors.New("invalid reissuable token param")
		}
	} else if param.MaxSupply.Sign() > 0 {
		return errors.New("invalid token param")
	}
	return nil
}
func (p *MethodMint) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamMintage)
	cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameMint, sendBlock.Data)
	key := cabi.GetMintageKey(param.TokenId)
	if len(db.GetStorage(&block.AccountAddress, key)) > 0 {
		return nil, util.ErrIdCollision
	}
	tokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameTokenInfo,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		sendBlock.AccountAddress,
		param.IsReIssuable,
		param.MaxSupply,
		param.OwnerBurnOnly)
	db.SetStorage(key, tokenInfo)
	if param.IsReIssuable {
		ownerTokenIdListKey := cabi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
		oldIdList := db.GetStorage(&block.AccountAddress, ownerTokenIdListKey)
		db.SetStorage(ownerTokenIdListKey, cabi.AppendTokenId(oldIdList, param.TokenId))
	}

	db.AddLog(util.NewLog(cabi.ABIMintage, cabi.EventNameMint, param.TokenId))
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
	return nil, nil
}

type MethodIssue struct{}

func (p *MethodIssue) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodIssue) GetRefundData() []byte {
	return []byte{4}
}
func (p *MethodIssue) GetQuota() uint64 {
	return IssueGas
}
func (p *MethodIssue) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if !fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	param := new(cabi.ParamIssue)
	err = cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameIssue, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if param.Amount.Sign() <= 0 || block.Amount.Sign() > 0 {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	tokenInfo := cabi.GetTokenById(db, param.TokenId)
	if tokenInfo == nil || !tokenInfo.IsReIssuable || tokenInfo.Owner != block.AccountAddress ||
		new(big.Int).Sub(tokenInfo.MaxSupply, tokenInfo.TotalSupply).Cmp(param.Amount) < 0 {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(cabi.MethodNameIssue, param.TokenId, param.Amount, param.Beneficial)
	return quotaLeft, nil
}
func (p *MethodIssue) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamIssue)
	cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameIssue, sendBlock.Data)
	oldTokenInfo := cabi.GetTokenById(db, param.TokenId)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress ||
		new(big.Int).Sub(oldTokenInfo.MaxSupply, oldTokenInfo.TotalSupply).Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Add(oldTokenInfo.TotalSupply, param.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	db.SetStorage(cabi.GetMintageKey(param.TokenId), newTokenInfo)

	db.AddLog(util.NewLog(cabi.ABIMintage, cabi.EventNameIssue, param.TokenId))
	return []*SendBlock{
		{
			block,
			param.Beneficial,
			ledger.BlockTypeSendReward,
			param.Amount,
			param.TokenId,
			[]byte{},
		},
	}, nil
}

type MethodBurn struct{}

func (p *MethodBurn) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodBurn) GetRefundData() []byte {
	return []byte{5}
}
func (p *MethodBurn) GetQuota() uint64 {
	return BurnGas
}
func (p *MethodBurn) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if !fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() <= 0 {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	tokenInfo := cabi.GetTokenById(db, block.TokenId)
	if tokenInfo == nil || !tokenInfo.IsReIssuable ||
		(tokenInfo.OwnerBurnOnly && tokenInfo.Owner != block.AccountAddress) {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(cabi.MethodNameBurn)
	return quotaLeft, nil
}
func (p *MethodBurn) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	oldTokenInfo := cabi.GetTokenById(db, sendBlock.TokenId)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable ||
		(oldTokenInfo.OwnerBurnOnly && oldTokenInfo.Owner != sendBlock.AccountAddress) {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Sub(oldTokenInfo.TotalSupply, sendBlock.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	db.SubBalance(&sendBlock.TokenId, sendBlock.Amount)
	db.SetStorage(cabi.GetMintageKey(sendBlock.TokenId), newTokenInfo)

	db.AddLog(util.NewLog(cabi.ABIMintage, cabi.EventNameBurn, sendBlock.TokenId, sendBlock.AccountAddress, sendBlock.Amount))
	return nil, nil
}

type MethodTransferOwner struct{}

func (p *MethodTransferOwner) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodTransferOwner) GetRefundData() []byte {
	return []byte{6}
}
func (p *MethodTransferOwner) GetQuota() uint64 {
	return TransferOwnerGas
}
func (p *MethodTransferOwner) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if !fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() > 0 {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	param := new(cabi.ParamTransferOwner)
	err = cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameTransferOwner, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if param.NewOwner == block.AccountAddress {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	tokenInfo := cabi.GetTokenById(db, param.TokenId)
	if tokenInfo == nil || !tokenInfo.IsReIssuable || tokenInfo.Owner != block.AccountAddress {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(cabi.MethodNameTransferOwner, param.TokenId, param.NewOwner)
	return quotaLeft, nil
}
func (p *MethodTransferOwner) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamTransferOwner)
	cabi.ABIMintage.UnpackMethod(param, cabi.MethodNameTransferOwner, sendBlock.Data)
	oldTokenInfo := cabi.GetTokenById(db, param.TokenId)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply,
		oldTokenInfo.Decimals,
		param.NewOwner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly)
	db.SetStorage(cabi.GetMintageKey(param.TokenId), newTokenInfo)

	oldKey := cabi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList := db.GetStorage(&block.AccountAddress, oldKey)
	db.SetStorage(oldKey, cabi.DeleteTokenId(oldIdList, param.TokenId))
	newKey := cabi.GetOwnerTokenIdListKey(param.NewOwner)
	newIdList := db.GetStorage(&block.AccountAddress, newKey)
	db.SetStorage(newKey, cabi.AppendTokenId(newIdList, param.TokenId))

	db.AddLog(util.NewLog(cabi.ABIMintage, cabi.EventNameTransferOwner, param.TokenId, param.NewOwner))
	return nil, nil
}

type MethodChangeTokenType struct{}

func (p *MethodChangeTokenType) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodChangeTokenType) GetRefundData() []byte {
	return []byte{7}
}
func (p *MethodChangeTokenType) GetQuota() uint64 {
	return ChangeTokenTypeGas
}
func (p *MethodChangeTokenType) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if !fork.IsMintFork(db.CurrentSnapshotBlock().Height) {
		return quotaLeft, util.ErrVersionNotSupport
	}
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	tokenId := new(types.TokenTypeId)
	err = cabi.ABIMintage.UnpackMethod(tokenId, cabi.MethodNameChangeTokenType, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if tokenId == nil || block.Amount.Sign() > 0 {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	tokenInfo := cabi.GetTokenById(db, *tokenId)
	if tokenInfo == nil || !tokenInfo.IsReIssuable || tokenInfo.Owner != block.AccountAddress {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	block.Data, _ = cabi.ABIMintage.PackMethod(cabi.MethodNameChangeTokenType, &tokenId)
	return quotaLeft, nil
}
func (p *MethodChangeTokenType) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	tokenId := new(types.TokenTypeId)
	cabi.ABIMintage.UnpackMethod(tokenId, cabi.MethodNameChangeTokenType, sendBlock.Data)
	oldTokenInfo := cabi.GetTokenById(db, *tokenId)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := cabi.ABIMintage.PackVariable(
		cabi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply,
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		false,
		helper.Big0,
		false)
	db.SetStorage(cabi.GetMintageKey(*tokenId), newTokenInfo)

	oldKey := cabi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList := db.GetStorage(&block.AccountAddress, oldKey)
	db.SetStorage(oldKey, cabi.DeleteTokenId(oldIdList, *tokenId))

	db.AddLog(util.NewLog(cabi.ABIMintage, cabi.EventNameChangeTokenType, *tokenId))
	return nil, nil
}
