package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const (
	jsonMintage = `
	[
		{"type":"function","name":"Mintage","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"}]},
		{"type":"function","name":"CancelPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]}
	]`

	MethodNameMintage             = "Mintage"
	MethodNameMintageCancelPledge = "CancelPledge"
	VariableNameMintage           = "mintage"
)

var (
	ABIMintage, _ = abi.JSONToABIContract(strings.NewReader(jsonMintage))
)

type ParamMintage struct {
	TokenId     types.TokenTypeId
	TokenName   string
	TokenSymbol string
	TotalSupply *big.Int
	Decimals    uint8
}

type TokenInfo struct {
	TokenName      string        `json:"tokenName"`
	TokenSymbol    string        `json:"tokenSymbol"`
	TotalSupply    *big.Int      `json:"totalSupply"`
	Decimals       uint8         `json:"decimals"`
	Owner          types.Address `json:"owner"`
	PledgeAmount   *big.Int      `json:"pledgeAmount"`
	WithdrawHeight uint64        `json:"withdrawHeight"`
}

func GetMintageKey(tokenId types.TokenTypeId) []byte {
	return tokenId.Bytes()
}
func GetTokenIdFromMintageKey(key []byte) types.TokenTypeId {
	tokenId, _ := types.BytesToTokenTypeId(key)
	return tokenId
}

func NewTokenId(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetTokenById(db StorageDatabase, tokenId types.TokenTypeId) *TokenInfo {
	data := db.GetStorage(&AddressMintage, GetMintageKey(tokenId))
	if len(data) > 0 {
		tokenInfo := new(TokenInfo)
		ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db StorageDatabase) map[types.TokenTypeId]*TokenInfo {
	defer monitor.LogTime("vm", "GetTokenMap", time.Now())
	iterator := db.NewStorageIterator(&AddressMintage, nil)
	tokenInfoMap := make(map[types.TokenTypeId]*TokenInfo)
	if iterator == nil {
		return tokenInfoMap
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		tokenId := GetTokenIdFromMintageKey(key)
		tokenInfo := new(TokenInfo)
		if err := ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, value); err == nil {
			tokenInfoMap[tokenId] = tokenInfo
		}
	}
	return tokenInfoMap
}

type MethodMintage struct{}

func (p *MethodMintage) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	if block.AccountBlock.Amount.Cmp(mintagePledgeAmount) == 0 && util.IsViteToken(block.AccountBlock.TokenId) {
		// Pledge ViteToken to mintage
		return big.NewInt(0), nil
	} else if block.AccountBlock.Amount.Sign() > 0 {
		return big.NewInt(0), errors.New("invalid amount")
	}
	// Destroy ViteToken to mintage
	return new(big.Int).Set(mintageFee), nil
}

func (p *MethodMintage) GetRefundData() []byte {
	return []byte{1}
}

func (p *MethodMintage) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, MintageGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(ParamMintage)
	err = ABIMintage.UnpackMethod(param, MethodNameMintage, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = CheckToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := NewTokenId(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.SnapshotHash)
	if GetTokenById(block.VmContext, tokenId) != nil {
		return quotaLeft, util.ErrIdCollision
	}
	block.AccountBlock.Data, _ = ABIMintage.PackMethod(
		MethodNameMintage,
		tokenId,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals)
	return quotaLeft, nil
}
func CheckToken(param ParamMintage) error {
	if param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		param.TotalSupply.Cmp(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(param.Decimals)), nil)) < 0 ||
		len(param.TokenName) == 0 || len(param.TokenName) > tokenNameLengthMax ||
		len(param.TokenSymbol) == 0 || len(param.TokenSymbol) > tokenSymbolLengthMax {
		return errors.New("invalid token param")
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", param.TokenName); !ok {
		return errors.New("invalid token name")
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", param.TokenSymbol); !ok {
		return errors.New("invalid token symbol")
	}
	return nil
}
func (p *MethodMintage) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamMintage)
	ABIMintage.UnpackMethod(param, MethodNameMintage, sendBlock.Data)
	key := GetMintageKey(param.TokenId)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return util.ErrIdCollision
	}
	var tokenInfo []byte
	if sendBlock.Amount.Sign() == 0 {
		tokenInfo, _ = ABIMintage.PackVariable(
			VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			uint64(0))
	} else {
		tokenInfo, _ = ABIMintage.PackVariable(
			VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			block.VmContext.CurrentSnapshotBlock().Height+nodeConfig.params.MintagePledgeHeight)
	}
	block.VmContext.SetStorage(key, tokenInfo)
	context.AppendBlock(
		&vm_context.VmAccountBlock{
			util.MakeSendBlock(
				block.AccountBlock,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendReward,
				param.TotalSupply,
				param.TokenId,
				context.GetNewBlockHeight(block),
				[]byte{}),
			nil})
	return nil
}

type MethodMintageCancelPledge struct{}

func (p *MethodMintageCancelPledge) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodMintageCancelPledge) GetRefundData() []byte {
	return []byte{2}
}

func (p *MethodMintageCancelPledge) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, MintageCancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 {
		return quotaLeft, errors.New("invalid block data")
	}
	tokenId := new(types.TokenTypeId)
	if err = ABIMintage.UnpackMethod(tokenId, MethodNameMintageCancelPledge, block.AccountBlock.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	return quotaLeft, nil
}
func (p *MethodMintageCancelPledge) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	tokenId := new(types.TokenTypeId)
	ABIMintage.UnpackMethod(tokenId, MethodNameMintageCancelPledge, sendBlock.Data)
	storageKey := GetMintageKey(*tokenId)
	tokenInfo := new(TokenInfo)
	ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, storageKey))

	if tokenInfo.Owner != sendBlock.AccountAddress ||
		tokenInfo.PledgeAmount.Sign() == 0 ||
		tokenInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return errors.New("cannot withdraw mintage pledge, status error")
	}

	newTokenInfo, _ := ABIMintage.PackVariable(
		VariableNameMintage,
		tokenInfo.TokenName,
		tokenInfo.TokenSymbol,
		tokenInfo.TotalSupply,
		tokenInfo.Decimals,
		tokenInfo.Owner,
		big.NewInt(0),
		uint64(0))
	block.VmContext.SetStorage(storageKey, newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		context.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					tokenInfo.Owner,
					ledger.BlockTypeSendCall,
					tokenInfo.PledgeAmount,
					ledger.ViteTokenId,
					context.GetNewBlockHeight(block),
					[]byte{}),
				nil})
	}
	return nil
}
