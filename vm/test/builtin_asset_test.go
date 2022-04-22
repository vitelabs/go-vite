package test

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	abi2 "github.com/vitelabs/go-vite/v2/vm/abi"
	"github.com/vitelabs/go-vite/v2/vm/builtin"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/util"
	"math/big"
	"testing"
)

var viteTotalSupply = new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))

func TestBuiltinAssetContract(t *testing.T) {
	db, owner := InitTestContext(t)
	// issue
	balance1 := new(big.Int).Set(viteTotalSupply)
	isReIssuable := true
	tokenName := "My Coin"
	tokenSymbol := "MYC"
	totalSupply := big.NewInt(1e10)
	maxSupply := new(big.Int).Mul(big.NewInt(2), totalSupply)
	decimals := uint8(3)

	fee := new(big.Int).Mul(big.NewInt(1e3), util.AttovPerVite)
	balance1.Sub(balance1, fee)

	calldata, _ := builtin.ContractAssetV3.Abi.PackMethod("issue", isReIssuable, tokenName, tokenSymbol, totalSupply, decimals, maxSupply)

	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	triggered := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, triggered)
	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
	assert.Equal(t, totalSupply, triggered.Amount)
	tokenId := triggered.TokenId
	tokenInfo, err := abi.GetTokenByID(db, tokenId)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenInfo)
	assert.Equal(t, tokenName, tokenInfo.TokenName)
	assert.Equal(t, tokenSymbol, tokenInfo.TokenSymbol)
	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
	assert.Equal(t, maxSupply, tokenInfo.MaxSupply)
	assert.Equal(t, decimals, tokenInfo.Decimals)
	// check events
	logs := vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	assert.Equal(t, 3, len(logs[0].Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["Issue"].Id(), logs[0].Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), logs[0].Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(owner.Bytes(), 32), logs[0].Topics[2].Bytes())

	// mint
	mintAmount := big.NewInt(1000)
	mintAddress, err := types.HexToAddress("vite_afc922b148b3b792ecff2e79fa17255c22f15d43a77dd79f15")
	assert.NoError(t, err)
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("mint", tokenId, mintAddress, mintAmount)
	assert.NoError(t, err)
	vmBlock, err = callContract(CallParams{BlockType: ledger.BlockTypeSendSyncCall, Callback: big.NewInt(123), Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)

	mintBlock := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, mintBlock)
	assert.Equal(t, ledger.BlockTypeSendReward, mintBlock.BlockType)
	assert.Equal(t, mintAddress, mintBlock.ToAddress)
	assert.Equal(t, tokenId, mintBlock.TokenId)
	assert.Equal(t, mintAmount, mintBlock.Amount)
	ret := vmBlock.AccountBlock.SendBlockList[1].Data
	b := new([32]byte)
	err = builtin.ContractAssetV3.Abi.Methods["mint"].Outputs.Unpack(b, ret[4:])
	assert.NoError(t, err)
	h, err := types.BytesToHash((*b)[:])
	assert.NoError(t, err)
	assert.Equal(t, mintBlock.Hash, h)

	tokenInfo, err = abi.GetTokenByID(db, tokenId)
	assert.NoError(t, err)
	totalSupply = big.NewInt(0).Add(totalSupply, mintAmount)
	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
	// check events
	logs = vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	assert.Equal(t, 3, len(logs[1].Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["Mint"].Id(), logs[1].Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), logs[1].Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(mintAddress.Bytes(), 32), logs[1].Topics[2].Bytes())
	assert.Equal(t, helper.LeftPadBytes(mintAmount.Bytes(), 32), logs[1].Data)

	// burn
	burnAmount := big.NewInt(500)
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("burn")
	assert.NoError(t, err)
	vmBlock, err = callContract(CallParams{BlockType: ledger.BlockTypeSendSyncCall, Callback: big.NewInt(123), Address: types.AddressAsset, Calldata: calldata, TokenId: &tokenId, Amount: burnAmount})
	assert.NoError(t, err)
	assert.NotEmpty(t, vmBlock.AccountBlock.SendBlockList)
	tokenInfo, err = abi.GetTokenByID(db, tokenId)
	assert.NoError(t, err)
	totalSupply = big.NewInt(0).Sub(totalSupply, burnAmount)
	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)

	ret = vmBlock.AccountBlock.SendBlockList[0].Data
	isOK := new(bool)
	err = builtin.ContractAssetV3.Abi.Methods["burn"].Outputs.Unpack(isOK, ret[4:])
	assert.NoError(t, err)
	assert.Equal(t, true, *isOK)
	// check events
	logs = vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	assert.Equal(t, 2, len(logs[2].Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["Burn"].Id(), logs[2].Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), logs[2].Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(burnAmount.Bytes(), 32), logs[2].Data)

	// name
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("name", tokenId)
	assert.NoError(t, err)

	vmBlock, err = callContract(CallParams{BlockType: ledger.BlockTypeSendSyncCall, Address: types.AddressAsset, Calldata: calldata, Callback: big.NewInt(123)})
	assert.NoError(t, err)

	ret = vmBlock.AccountBlock.SendBlockList[0].Data
	gotName := new(string)
	builtin.ContractAssetV3.Abi.Methods["name"].Outputs.Unpack(gotName, ret[4:])
	assert.NotEmpty(t, gotName)
	assert.Equal(t, "My Coin", *gotName)
}

func TestIssueVEP19(t *testing.T) {
	db, _ := InitTestContext(t)

	// issue by BEP19
	calldata, err := builtin.ContractAssetV3.Abi.PackMethod("issue", true, "My Coin2", "MYC2", big.NewInt(2e10), uint8(2), big.NewInt(2e10))
	assert.NoError(t, err)

	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata, Callback: big.NewInt(123)})
	assert.NoError(t, err)

	assert.NotEmpty(t, vmBlock.AccountBlock.SendBlockList)
	assert.Equal(t, 2, len(vmBlock.AccountBlock.SendBlockList))

	mintBlock := vmBlock.AccountBlock.SendBlockList[0]
	callbackBlock := vmBlock.AccountBlock.SendBlockList[1]

	assert.Equal(t, ledger.BlockTypeSendReward, mintBlock.BlockType)
	assert.Equal(t, ledger.BlockTypeSendCallback, callbackBlock.BlockType)

	ret := callbackBlock.Data
	assert.Equal(t, big.NewInt(123), big.NewInt(0).SetBytes(ret[:4]))
	tokenId := new(types.TokenTypeId)
	err = builtin.ContractAssetV3.Abi.Methods["issue"].Outputs.Unpack(tokenId, ret[4:])
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenId)

	tokenInfo, err := abi.GetTokenByID(db, *tokenId)
	assert.NoError(t, err)
	assert.Equal(t, "My Coin2", tokenInfo.TokenName)
	assert.Equal(t, "MYC2", tokenInfo.TokenSymbol)
	assert.Equal(t, big.NewInt(2e10), tokenInfo.TotalSupply)
	assert.Equal(t, big.NewInt(2e10), tokenInfo.MaxSupply)
	assert.Equal(t, uint8(2), tokenInfo.Decimals)

	context, err := db.GetExecutionContext(&callbackBlock.Hash)
	assert.NoError(t, err)
	assert.NotEmpty(t, context)
	sendBlockHash := vmBlock.AccountBlock.FromBlockHash
	assert.Equal(t, sendBlockHash, context.ReferrerSendHash)
}

func TestOwnership(t *testing.T) {
	db, issuer := InitTestContext(t)
	// issue
	calldata, err := builtin.ContractAssetV3.Abi.PackMethod("issue", true, "My Coin", "MYC", big.NewInt(1000), uint8(1), big.NewInt(2000))
	assert.NoError(t, err)
	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	triggered := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, triggered)
	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
	tokenId := triggered.TokenId

	// query owner
	queryData, err := builtin.ContractAssetV3.Abi.PackMethod("owner", tokenId)
	assert.NoError(t, err)
	ret, err := builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	owner := new(types.Address)
	err = builtin.ContractAssetV3.Abi.Methods["owner"].Outputs.Unpack(owner, ret)
	assert.NoError(t, err)
	assert.Equal(t, issuer, *owner)

	// mint by Alice
	alice, _ := types.HexToAddress("vite_afc922b148b3b792ecff2e79fa17255c22f15d43a77dd79f15")
	mintAmount := big.NewInt(100)
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("mint", tokenId, alice, mintAmount)
	assert.NoError(t, err)
	vmBlock, err = callContractBy(&alice, CallParams{BlockType: ledger.BlockTypeSendSyncCall, Callback: big.NewInt(123), Address: types.AddressAsset, Calldata: calldata})
	assert.Equal(t, util.ErrAccessDenied, err)

	// transfer ownership by Alice
	bob, _ := types.HexToAddress("vite_629ff97c21394b9c0ca6ccec4ff6e54954fca8db27397cc209")
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("transferOwnership", tokenId, bob)
	assert.NoError(t, err)
	vmBlock, err = callContractBy(&alice, CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.Equal(t, util.ErrAccessDenied, err)

	// transfer ownership to Alice
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("transferOwnership", tokenId, alice)
	assert.NoError(t, err)
	vmBlock, err = callContractBy(&issuer, CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	// check events
	logs := vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	ownerTransferEvent := logs[1]
	assert.Equal(t, 4, len(ownerTransferEvent.Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["OwnershipTransferred"].Id(), ownerTransferEvent.Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), ownerTransferEvent.Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(issuer.Bytes(), 32), ownerTransferEvent.Topics[2].Bytes())
	assert.Equal(t, helper.LeftPadBytes(alice.Bytes(), 32), ownerTransferEvent.Topics[3].Bytes())

	// mint by Alice
	calldata, err = builtin.ContractAssetV3.Abi.PackMethod("mint", tokenId, bob, mintAmount)
	assert.NoError(t, err)
	vmBlock, err = callContractBy(&alice, CallParams{BlockType: ledger.BlockTypeSendSyncCall, Callback: big.NewInt(123), Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	mintBlock := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, mintBlock)
	assert.Equal(t, ledger.BlockTypeSendReward, mintBlock.BlockType)
	assert.Equal(t, bob, mintBlock.ToAddress)
	assert.Equal(t, tokenId, mintBlock.TokenId)
	assert.Equal(t, mintAmount, mintBlock.Amount)
	ret = vmBlock.AccountBlock.SendBlockList[1].Data
	b := new([32]byte)
	err = builtin.ContractAssetV3.Abi.Methods["mint"].Outputs.Unpack(b, ret[4:])
	assert.NoError(t, err)
	h, err := types.BytesToHash((*b)[:])
	assert.NoError(t, err)
	assert.Equal(t, mintBlock.Hash, h)

	tokenInfo, err := abi.GetTokenByID(db, tokenId)
	assert.NoError(t, err)
	totalSupply := big.NewInt(1100)
	assert.Equal(t, totalSupply, tokenInfo.TotalSupply)
	// check events
	logs = vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	mintEvent := logs[2]
	assert.Equal(t, 3, len(mintEvent.Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["Mint"].Id(), mintEvent.Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), mintEvent.Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(bob.Bytes(), 32), mintEvent.Topics[2].Bytes())
	assert.Equal(t, helper.LeftPadBytes(mintAmount.Bytes(), 32), mintEvent.Data)

	// query owner
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("owner", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	err = builtin.ContractAssetV3.Abi.Methods["owner"].Outputs.Unpack(owner, ret)
	assert.NoError(t, err)
	assert.Equal(t, alice, *owner)
}

func TestQuery(t *testing.T) {
	db, caller := InitTestContext(t)
	// issue
	calldata, err := builtin.ContractAssetV3.Abi.PackMethod("issue", true, "My Coin", "MYC", big.NewInt(1e10), uint8(2), big.NewInt(2e10))
	assert.NoError(t, err)
	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	triggered := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, triggered)
	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
	tokenId := triggered.TokenId

	// name
	queryData, err := builtin.ContractAssetV3.Abi.PackMethod("name", tokenId)
	assert.NoError(t, err)
	ret, err := builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	gotName := new(string)
	builtin.ContractAssetV3.Abi.Methods["name"].Outputs.Unpack(gotName, ret)
	assert.NotEmpty(t, gotName)
	assert.Equal(t, "My Coin", *gotName)

	// symbol
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("symbol", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	symbol := new(string)
	builtin.ContractAssetV3.Abi.Methods["symbol"].Outputs.Unpack(symbol, ret)
	assert.Equal(t, "MYC", *symbol)

	// decimals
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("decimals", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	decimals := new(uint8)
	builtin.ContractAssetV3.Abi.Methods["decimals"].Outputs.Unpack(decimals, ret)
	assert.Equal(t,  uint8(2), *decimals)

	// totalSupply
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("totalSupply", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	totalSupply, err := builtin.ContractAssetV3.Abi.Methods["totalSupply"].Outputs.DirectUnpack(ret)
	assert.NoError(t, err)
	assert.Equal(t,  big.NewInt(1e10), totalSupply[0].(*big.Int))

	// maxSupply
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("maxSupply", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	maxSupply, err := builtin.ContractAssetV3.Abi.Methods["maxSupply"].Outputs.DirectUnpack(ret)
	assert.NoError(t, err)
	assert.Equal(t,  big.NewInt(2e10), maxSupply[0].(*big.Int))

	// mintable
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("mintable", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	mintable := new(bool)
	err = builtin.ContractAssetV3.Abi.Methods["mintable"].Outputs.Unpack(mintable, ret)
	assert.NoError(t, err)
	assert.Equal(t, true, *mintable)

	// owner
	queryData, err = builtin.ContractAssetV3.Abi.PackMethod("owner", tokenId)
	assert.NoError(t, err)
	ret, err = builtin.Query(db, types.AddressAsset, queryData)
	assert.NoError(t, err)
	owner := new(types.Address)
	err = builtin.ContractAssetV3.Abi.Methods["owner"].Outputs.Unpack(owner, ret)
	assert.NoError(t, err)
	assert.Equal(t, caller, *owner)
}

func TestError(t *testing.T) {
	InitTestContext(t)

	// issue by BEP19
	calldata, err := builtin.ContractAssetV3.Abi.PackMethod("issue", false, "My Coin2", "MYC2", big.NewInt(0), uint8(2), big.NewInt(0))
	assert.NoError(t, err)

	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata, Callback: big.NewInt(123)})
	assert.Error(t, err)
	assert.Equal(t, util.ErrInvalidMethodParam, err)

	assert.NotEmpty(t, vmBlock.AccountBlock.SendBlockList)
	assert.Equal(t, 2, len(vmBlock.AccountBlock.SendBlockList))

	refundBlock := vmBlock.AccountBlock.SendBlockList[0]
	callbackBlock := vmBlock.AccountBlock.SendBlockList[1]

	assert.Equal(t, ledger.BlockTypeSendCall, refundBlock.BlockType)
	assert.Equal(t, ledger.BlockTypeSendFailureCallback, callbackBlock.BlockType)

	ret := callbackBlock.Data
	assert.Equal(t, big.NewInt(123), big.NewInt(0).SetBytes(ret[:4]))

	e, _ := abi2.PackError("invalid method param")
	assert.Equal(t, hex.EncodeToString(e), hex.EncodeToString(ret[4:]))

	context, err := testDB.GetExecutionContext(&callbackBlock.Hash)
	assert.NoError(t, err)
	assert.NotEmpty(t, context)
	sendBlockHash := vmBlock.AccountBlock.FromBlockHash
	assert.Equal(t, sendBlockHash, context.ReferrerSendHash)
}

func TestReceive(t *testing.T) {
	db, _ := InitTestContext(t)

	calldata, err := builtin.ContractAssetV3.Abi.PackMethod("issue", true, "My Coin", "MYC", big.NewInt(1000), uint8(1), big.NewInt(2000))
	assert.NoError(t, err)
	vmBlock, err := callContract(CallParams{Address: types.AddressAsset, Calldata: calldata})
	assert.NoError(t, err)
	triggered := vmBlock.AccountBlock.SendBlockList[0]
	assert.NotEmpty(t, triggered)
	assert.Equal(t, ledger.BlockTypeSendReward, triggered.BlockType)
	tokenId := triggered.TokenId

	// send block without data to burn tokens
	vmBlock, err = callContract(CallParams{Address: types.AddressAsset, Calldata: []byte{}, TokenId: &tokenId, Amount: big.NewInt(100)})
	assert.NoError(t, err)
	tokenInfo, err := abi.GetTokenByID(db, tokenId)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(900), tokenInfo.TotalSupply)

	// check events
	logs := vmBlock.VmDb.GetLogList()
	assert.NotEmpty(t, logs)
	event := logs[1]
	assert.Equal(t, 2, len(event.Topics))
	assert.Equal(t, builtin.ContractAssetV3.Abi.Events["Burn"].Id(), event.Topics[0])
	assert.Equal(t, helper.LeftPadBytes(tokenId.Bytes(), 32), event.Topics[1].Bytes())
	assert.Equal(t, helper.LeftPadBytes(big.NewInt(100).Bytes(), 32), event.Data)
}