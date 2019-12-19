package contracts

import (
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"regexp"
)

type MethodIssue struct {
	MethodName string
}

func (p *MethodIssue) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	if block.Amount.Sign() > 0 {
		return big.NewInt(0), util.ErrInvalidMethodParam
	}
	return new(big.Int).Set(issueFee), nil
}
func (p *MethodIssue) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodIssue) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.IssueQuota, nil
}
func (p *MethodIssue) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodIssue) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamIssue)
	err := abi.ABIAsset.UnpackMethod(param, p.MethodName, block.Data)
	if err != nil {
		return err
	}
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	if err = checkToken(*param, sb.Height); err != nil {
		return err
	}
	block.Data, _ = abi.ABIAsset.PackMethod(
		p.MethodName,
		param.IsReIssuable,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		param.MaxSupply,
		param.IsOwnerBurnOnly)
	return nil
}

func checkToken(param abi.ParamIssue, sbHeight uint64) error {
	if param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		len(param.TokenName) == 0 || len(param.TokenName) > tokenNameLengthMax ||
		len(param.TokenSymbol) == 0 || len(param.TokenSymbol) > tokenSymbolLengthMax {
		return util.ErrInvalidMethodParam
	}
	if !fork.IsEarthFork(sbHeight) && (param.TotalSupply.Sign() <= 0 ||
		param.TotalSupply.Cmp(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(param.Decimals)), nil)) < 0) {
		return util.ErrInvalidMethodParam
	}
	if fork.IsEarthFork(sbHeight) && !param.IsReIssuable && param.TotalSupply.Sign() <= 0 {
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
	} else if param.MaxSupply.Sign() > 0 || param.IsOwnerBurnOnly {
		return util.ErrInvalidMethodParam
	}
	return nil
}
func (p *MethodIssue) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamIssue)
	abi.ABIAsset.UnpackMethod(param, p.MethodName, sendBlock.Data)
	tokenID := newTokenID(sendBlock.AccountAddress, block.Height, sendBlock.Hash)
	key := abi.GetTokenInfoKey(tokenID)
	v := util.GetValue(db, key)
	if len(v) > 0 {
		return nil, util.ErrIDCollision
	}
	nextIndex := uint16(0)
	nextIndexKey := abi.GetNextTokenIndexKey(param.TokenSymbol)
	nextV := util.GetValue(db, nextIndexKey)
	if len(nextV) > 0 {
		nextIndexPtr := new(uint16)
		abi.ABIAsset.UnpackVariable(nextIndexPtr, abi.VariableNameTokenIndex, nextV)
		nextIndex = *nextIndexPtr
	}
	if nextIndex == tokenNameIndexMax {
		return nil, util.ErrInvalidMethodParam
	}

	ownerTokenIDListKey := abi.GetTokenIDListKey(sendBlock.AccountAddress)
	oldIDList := util.GetValue(db, ownerTokenIDListKey)

	tokenInfo, _ := abi.ABIAsset.PackVariable(
		abi.VariableNameTokenInfo,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals,
		sendBlock.AccountAddress,
		param.IsReIssuable,
		param.MaxSupply,
		param.IsOwnerBurnOnly,
		nextIndex)
	util.SetValue(db, key, tokenInfo)
	util.SetValue(db, ownerTokenIDListKey, abi.AppendTokenID(oldIDList, tokenID))
	nextV, _ = abi.ABIAsset.PackVariable(abi.VariableNameTokenIndex, nextIndex+1)
	util.SetValue(db, nextIndexKey, nextV)

	db.AddLog(NewLog(abi.ABIAsset, util.FirstToLower(p.MethodName), tokenID))
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendReward,
			Amount:         param.TotalSupply,
			TokenId:        tokenID,
			Data:           []byte{},
		},
	}, nil
	return nil, nil
}

func newTokenID(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		helper.LeftPadBytes(new(big.Int).SetUint64(accountBlockHeight).Bytes(), 8),
		prevBlockHash.Bytes())
}

type MethodReIssue struct {
	MethodName string
}

func (p *MethodReIssue) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodReIssue) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodReIssue) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.ReIssueQuota, nil
}
func (p *MethodReIssue) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodReIssue) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamReIssue)
	err := abi.ABIAsset.UnpackMethod(param, p.MethodName, block.Data)
	if err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName, param.TokenId, param.Amount, param.ReceiveAddress)
	return nil
}
func (p *MethodReIssue) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamReIssue)
	abi.ABIAsset.UnpackMethod(param, p.MethodName, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenByID(db, param.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress ||
		new(big.Int).Sub(oldTokenInfo.MaxSupply, oldTokenInfo.TotalSupply).Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIAsset.PackVariable(
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
	util.SetValue(db, abi.GetTokenInfoKey(param.TokenId), newTokenInfo)

	db.AddLog(NewLog(abi.ABIAsset, util.FirstToLower(p.MethodName), param.TokenId))
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      param.ReceiveAddress,
			BlockType:      ledger.BlockTypeSendReward,
			Amount:         param.Amount,
			TokenId:        param.TokenId,
			Data:           []byte{},
		},
	}, nil
}

type MethodBurn struct {
	MethodName string
}

func (p *MethodBurn) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodBurn) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodBurn) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.BurnQuota, nil
}
func (p *MethodBurn) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodBurn) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName)
	return nil
}
func (p *MethodBurn) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	oldTokenInfo, err := abi.GetTokenByID(db, sendBlock.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || (!util.CheckFork(db, fork.IsEarthFork) && !oldTokenInfo.IsReIssuable) ||
		(oldTokenInfo.OwnerBurnOnly && oldTokenInfo.Owner != sendBlock.AccountAddress) {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIAsset.PackVariable(
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
	util.SetValue(db, abi.GetTokenInfoKey(sendBlock.TokenId), newTokenInfo)

	db.AddLog(NewLog(abi.ABIAsset, util.FirstToLower(p.MethodName), sendBlock.TokenId, sendBlock.AccountAddress, sendBlock.Amount))
	return nil, nil
}

type MethodTransferOwnership struct {
	MethodName string
}

func (p *MethodTransferOwnership) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodTransferOwnership) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodTransferOwnership) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.TransferOwnershipQuota, nil
}
func (p *MethodTransferOwnership) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodTransferOwnership) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamTransferOwnership)
	err := abi.ABIAsset.UnpackMethod(param, p.MethodName, block.Data)
	if err != nil {
		return err
	}
	if param.NewOwner == block.AccountAddress {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName, param.TokenId, param.NewOwner)
	return nil
}
func (p *MethodTransferOwnership) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamTransferOwnership)
	abi.ABIAsset.UnpackMethod(param, p.MethodName, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenByID(db, param.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIAsset.PackVariable(
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
	util.SetValue(db, abi.GetTokenInfoKey(param.TokenId), newTokenInfo)

	oldKey := abi.GetTokenIDListKey(sendBlock.AccountAddress)
	oldIDList := util.GetValue(db, oldKey)
	util.SetValue(db, oldKey, abi.DeleteTokenID(oldIDList, param.TokenId))
	newKey := abi.GetTokenIDListKey(param.NewOwner)
	newIDList := util.GetValue(db, newKey)
	util.SetValue(db, newKey, abi.AppendTokenID(newIDList, param.TokenId))

	db.AddLog(NewLog(abi.ABIAsset, util.FirstToLower(p.MethodName), param.TokenId, param.NewOwner))
	return nil, nil
}

type MethodDisableReIssue struct {
	MethodName string
}

func (p *MethodDisableReIssue) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodDisableReIssue) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodDisableReIssue) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.DisableReIssueQuota, nil
}
func (p *MethodDisableReIssue) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodDisableReIssue) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	tokenID := new(types.TokenTypeId)
	err := abi.ABIAsset.UnpackMethod(tokenID, p.MethodName, block.Data)
	if err != nil {
		return err
	}
	if tokenID == nil || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName, &tokenID)
	return nil
}
func (p *MethodDisableReIssue) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	tokenID := new(types.TokenTypeId)
	abi.ABIAsset.UnpackMethod(tokenID, p.MethodName, sendBlock.Data)
	oldTokenInfo, err := abi.GetTokenByID(db, *tokenID)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable || oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}
	newTokenInfo, _ := abi.ABIAsset.PackVariable(
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
	util.SetValue(db, abi.GetTokenInfoKey(*tokenID), newTokenInfo)

	db.AddLog(NewLog(abi.ABIAsset, util.FirstToLower(p.MethodName), *tokenID))
	return nil, nil
}

type MethodGetTokenInfo struct {
	MethodName string
}

func (p *MethodGetTokenInfo) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodGetTokenInfo) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	param := new(abi.ParamGetTokenInfo)
	abi.ABIAsset.UnpackMethod(param, p.MethodName, sendBlock.Data)
	var callbackData []byte
	if p.MethodName == abi.MethodNameGetTokenInfoV3 {
		callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, sendBlock.Hash, param.TokenId, false, false, "", "", helper.Big0, uint8(0), helper.Big0, false, uint16(0), types.Address{})
	} else {
		callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, param.TokenId, param.Bid, false, uint8(0), "", uint16(0), types.Address{})
	}
	return callbackData, true
}
func (p *MethodGetTokenInfo) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.GetTokenInfoQuota, nil
}
func (p *MethodGetTokenInfo) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}
func (p *MethodGetTokenInfo) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamGetTokenInfo)
	err := abi.ABIAsset.UnpackMethod(param, p.MethodName, block.Data)
	if err != nil {
		return err
	}
	if param == nil || block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameGetTokenInfoV3 {
		block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName, param.TokenId)
	} else {
		block.Data, _ = abi.ABIAsset.PackMethod(p.MethodName, param.TokenId, param.Bid)
	}
	return nil
}

func (p *MethodGetTokenInfo) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamGetTokenInfo)
	abi.ABIAsset.UnpackMethod(param, p.MethodName, sendBlock.Data)
	tokenInfo, err := abi.GetTokenByID(db, param.TokenId)
	util.DealWithErr(err)
	var callbackData []byte
	if tokenInfo != nil {
		if p.MethodName == abi.MethodNameGetTokenInfoV3 {
			callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, sendBlock.Hash, param.TokenId, true, tokenInfo.IsReIssuable, tokenInfo.TokenName, tokenInfo.TokenSymbol, tokenInfo.TotalSupply, tokenInfo.Decimals, tokenInfo.MaxSupply, tokenInfo.OwnerBurnOnly, tokenInfo.Index, tokenInfo.Owner)
		} else {
			callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, param.TokenId, param.Bid, true, tokenInfo.Decimals, tokenInfo.TokenSymbol, tokenInfo.Index, tokenInfo.Owner)
		}
	} else {
		if p.MethodName == abi.MethodNameGetTokenInfoV3 {
			callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, sendBlock.Hash, param.TokenId, false, false, "", "", helper.Big0, uint8(0), helper.Big0, false, uint16(0), types.Address{})
		} else {
			callbackData, _ = abi.ABIAsset.PackCallback(p.MethodName, param.TokenId, param.Bid, false, uint8(0), "", uint16(0), types.Address{})
		}
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
