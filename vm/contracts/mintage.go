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
	if block.Amount.Sign() > 0 {
		return big.NewInt(0), util.ErrInvalidMethodParam
	}
	return new(big.Int).Set(mintageFee), nil
}
func (p *MethodMint) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodMint) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.MintGas, nil
}
func (p *MethodMint) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
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
	block.Data, _ = abi.ABIMintage.PackMethod(
		abi.MethodNameMint,
		param.IsReIssuable,
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
	if ok, _ := regexp.MatchString("^[A-Z0-9]+$", param.TokenSymbol); !ok {
		return util.ErrInvalidMethodParam
	}
	if param.TokenSymbol == "VITE" || param.TokenSymbol == "VCP" || param.TokenSymbol == "VX" {
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
	tokenId := abi.NewTokenId(sendBlock.AccountAddress, block.Height, sendBlock.Hash)
	key := abi.GetMintageKey(tokenId)
	v := util.GetValue(db, key)
	if len(v) > 0 {
		return nil, util.ErrIdCollision
	}
	nextIndex := uint16(0)
	nextIndexKey := abi.GetNextIndexKey(param.TokenSymbol)
	nextV := util.GetValue(db, nextIndexKey)
	if len(nextV) > 0 {
		nextIndexPtr := new(uint16)
		abi.ABIMintage.UnpackVariable(nextIndexPtr, abi.VariableNameTokenNameIndex, nextV)
		nextIndex = *nextIndexPtr
	}
	if nextIndex == tokenNameIndexMax {
		return nil, util.ErrInvalidMethodParam
	}

	ownerTokenIdListKey := abi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList := util.GetValue(db, ownerTokenIdListKey)

	tokenInfo, _ := abi.ABIMintage.PackVariable(
		abi.VariableNameTokenInfo,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		sendBlock.AccountAddress,
		param.IsReIssuable,
		param.MaxSupply,
		param.OwnerBurnOnly,
		nextIndex)
	util.SetValue(db, key, tokenInfo)
	util.SetValue(db, ownerTokenIdListKey, abi.AppendTokenId(oldIdList, tokenId))
	nextV, _ = abi.ABIMintage.PackVariable(abi.VariableNameTokenNameIndex, nextIndex+1)
	util.SetValue(db, nextIndexKey, nextV)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameMint, tokenId))
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendReward,
			Amount:         param.TotalSupply,
			TokenId:        tokenId,
			Data:           []byte{},
		},
	}, nil
	return nil, nil
}

type MethodIssue struct{}

func (p *MethodIssue) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodIssue) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodIssue) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.IssueGas, nil
}
func (p *MethodIssue) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
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
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)
	util.SetValue(db, abi.GetMintageKey(param.TokenId), newTokenInfo)

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
func (p *MethodBurn) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodBurn) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.BurnGas, nil
}
func (p *MethodBurn) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
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
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)
	util.SubBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	util.SetValue(db, abi.GetMintageKey(sendBlock.TokenId), newTokenInfo)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameBurn, sendBlock.TokenId, sendBlock.AccountAddress, sendBlock.Amount))
	return nil, nil
}

type MethodTransferOwner struct{}

func (p *MethodTransferOwner) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodTransferOwner) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodTransferOwner) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.TransferOwnerGas, nil
}
func (p *MethodTransferOwner) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
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
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)
	util.SetValue(db, abi.GetMintageKey(param.TokenId), newTokenInfo)

	oldKey := abi.GetOwnerTokenIdListKey(sendBlock.AccountAddress)
	oldIdList := util.GetValue(db, oldKey)
	util.SetValue(db, oldKey, abi.DeleteTokenId(oldIdList, param.TokenId))
	newKey := abi.GetOwnerTokenIdListKey(param.NewOwner)
	newIdList := util.GetValue(db, newKey)
	util.SetValue(db, newKey, abi.AppendTokenId(newIdList, param.TokenId))

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameTransferOwner, param.TokenId, param.NewOwner))
	return nil, nil
}

type MethodChangeTokenType struct{}

func (p *MethodChangeTokenType) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodChangeTokenType) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodChangeTokenType) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.ChangeTokenTypeGas, nil
}
func (p *MethodChangeTokenType) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
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
		false,
		helper.Big0,
		false,
		oldTokenInfo.Index)
	util.SetValue(db, abi.GetMintageKey(*tokenId), newTokenInfo)

	db.AddLog(util.NewLog(abi.ABIMintage, abi.EventNameChangeTokenType, *tokenId))
	return nil, nil
}

type MethodGetTokenInfo struct{}

func (p *MethodGetTokenInfo) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodGetTokenInfo) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	param := new(abi.ParamGetTokenInfo)
	abi.ABIMintage.UnpackMethod(param, abi.MethodNameGetTokenInfo, sendBlock.Data)
	callbackData, _ := abi.ABIMintage.PackCallback(abi.MethodNameGetTokenInfo, param.TokenId, param.Bid, false, uint8(0), "", uint16(0), types.Address{})
	return callbackData, true
}
func (p *MethodGetTokenInfo) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.GetTokenInfoGas, nil
}
func (p *MethodGetTokenInfo) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}
func (p *MethodGetTokenInfo) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamGetTokenInfo)
	err := abi.ABIMintage.UnpackMethod(param, abi.MethodNameGetTokenInfo, block.Data)
	if err != nil {
		return err
	}
	if param == nil || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIMintage.PackMethod(abi.MethodNameGetTokenInfo, param.TokenId, param.Bid)
	return nil
}

func (p *MethodGetTokenInfo) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamGetTokenInfo)
	abi.ABIMintage.UnpackMethod(param, abi.MethodNameGetTokenInfo, sendBlock.Data)
	tokenInfo, err := abi.GetTokenById(db, param.TokenId)
	util.DealWithErr(err)
	var callbackData []byte
	if tokenInfo != nil {
		callbackData, _ = abi.ABIMintage.PackCallback(abi.MethodNameGetTokenInfo, param.TokenId, param.Bid, true, tokenInfo.Decimals, tokenInfo.TokenSymbol, tokenInfo.Index, tokenInfo.Owner)
	} else {
		callbackData, _ = abi.ABIMintage.PackCallback(abi.MethodNameGetTokenInfo, param.TokenId, param.Bid, false, uint8(0), "", uint16(0), types.Address{})
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         big.NewInt(0),
			TokenId:        ledger.ViteTokenId,
			Data:           callbackData,
		},
	}, nil
}
