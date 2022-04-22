package builtin

import (
	"bytes"
	"encoding/binary"
	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm/abi"
	"github.com/vitelabs/go-vite/v2/vm/contracts"
	contractsAbi "github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/util"
	"math/big"
	"regexp"
)

const AbiAssetJson = `
[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "amount",
        "type": "uint256"
      }
    ],
    "name": "Burn",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "owner",
        "type": "address"
      }
    ],
    "name": "Issue",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "to",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "amount",
        "type": "uint256"
      }
    ],
    "name": "Mint",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "previousOwner",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "OwnershipTransferred",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "burn",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "decimals",
    "outputs": [
      {
        "internalType": "uint8",
        "name": "",
        "type": "uint8"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bool",
        "name": "_mintable",
        "type": "bool"
      },
      {
        "internalType": "string",
        "name": "_name",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "_symbol",
        "type": "string"
      },
      {
        "internalType": "uint256",
        "name": "_totalSupply",
        "type": "uint256"
      },
      {
        "internalType": "uint8",
        "name": "_decimals",
        "type": "uint8"
      },
      {
        "internalType": "uint256",
        "name": "_maxSupply",
        "type": "uint256"
      }
    ],
    "name": "issue",
    "outputs": [
      {
        "internalType": "tokenId",
        "name": "",
        "type": "tokenId"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "maxSupply",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "internalType": "address",
        "name": "to",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "amount",
        "type": "uint256"
      }
    ],
    "name": "mint",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "mintBlockHash",
        "type": "bytes32"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "mintable",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "name",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "owner",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "symbol",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      }
    ],
    "name": "totalSupply",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "tokenId",
        "name": "tokenId",
        "type": "tokenId"
      },
      {
        "internalType": "address",
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "transferOwnership",
    "outputs": [
      {
        "internalType": "bool",
        "name": "",
        "type": "bool"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`

var AbiAsset = abi.ParseABI(AbiAssetJson)

type ParamsIssue struct {
	Mintable    bool
	Name       string
	Symbol     string
	TotalSupply     *big.Int
	Decimals        uint8
	MaxSupply       *big.Int
}

type ParamMint struct {
	TokenId types.TokenTypeId
	To      types.Address
	Amount  *big.Int
}

type ParamTransferOwnership struct {
	TokenId 	types.TokenTypeId
	NewOwner 	types.Address
}

func NewTokenContract() *NativeContract {
	c := &NativeContract{
		Name: "Asset",
		Abi: &AbiAsset,
	}

	// initialize method map
	methods := make(map[uint32]*NativeContractMethod)

	methods[c.Abi.Methods["issue"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["issue"],
		Execute: issue,
	}
	methods[c.Abi.Methods["mint"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["mint"],
		Execute: mint,
	}
	methods[c.Abi.Methods["burn"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["burn"],
		Execute: burn,
	}
	methods[c.Abi.Methods["transferOwnership"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["transferOwnership"],
		Execute: transferOwnership,
	}
	methods[c.Abi.Methods["name"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["name"],
		Execute: name,
	}
	methods[c.Abi.Methods["symbol"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["symbol"],
		Execute: symbol,
	}
	methods[c.Abi.Methods["decimals"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["decimals"],
		Execute: decimals,
	}
	methods[c.Abi.Methods["totalSupply"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["totalSupply"],
		Execute: totalSupply,
	}
	methods[c.Abi.Methods["maxSupply"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["maxSupply"],
		Execute: maxSupply,
	}
	methods[c.Abi.Methods["mintable"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["mintable"],
		Execute: mintable,
	}
	methods[c.Abi.Methods["owner"].IdUint()] = &NativeContractMethod{
		Abi: c.Abi.Methods["owner"],
		Execute: owner,
	}

	// burn tokens in receive() function
	c.ReceiveFunction = burn

	c.Methods = methods
	return c
}

func issue(request NativeContractRequest) (*NativeContractResult, error) {
	sendBlock := request.SendBlock
	receiveBlock := request.ReceiveBlock
	methodAbi := request.Abi
	data := sendBlock.Data

	params := new(ParamsIssue)
	err := methodAbi.Inputs.Unpack(params, data[4:])
	if err != nil {
		return nil, err
	}
	// check params
	if params.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		len(params.Name) == 0 || len(params.Name) > contracts.TokenNameLengthMax ||
		len(params.Symbol) == 0 || len(params.Symbol) > contracts.TokenSymbolLengthMax {
		return nil, util.ErrInvalidMethodParam
	}

	if !params.Mintable && params.TotalSupply.Sign() <= 0 {
		return nil, util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", params.Name); !ok {
		return nil, util.ErrInvalidMethodParam
	}
	if ok, _ := regexp.MatchString("^[A-Z0-9]+$", params.Symbol); !ok {
		return nil, util.ErrInvalidMethodParam
	}
	if params.Symbol == "VITE" || params.Symbol == "VCP" || params.Symbol == "VX" {
		return nil, util.ErrInvalidMethodParam
	}
	if params.Mintable {
		if params.MaxSupply.Cmp(params.TotalSupply) < 0 || params.MaxSupply.Cmp(helper.Tt256m1) > 0 {
			return nil, util.ErrInvalidMethodParam
		}
	} else if params.MaxSupply.Sign() > 0 {
		return nil, util.ErrInvalidMethodParam
	}

	// generate token Id
	db := *request.Db
	tokenID := types.CreateTokenTypeId(
		sendBlock.AccountAddress.Bytes(),
		helper.LeftPadBytes(new(big.Int).SetUint64(receiveBlock.Height).Bytes(), 8),
		sendBlock.Hash.Bytes())
	key := contractsAbi.GetTokenInfoKey(tokenID)
	v := util.GetValue(db, key)
	if len(v) > 0 {
		return nil, util.ErrIDCollision
	}
	nextIndex := uint16(0)
	nextIndexKey := contractsAbi.GetNextTokenIndexKey(params.Symbol)
	nextV := util.GetValue(db, nextIndexKey)
	if len(nextV) > 0 {
		nextIndex = binary.BigEndian.Uint16(nextV[len(nextV)-2:])
	}
	if nextIndex == contracts.TokenNameIndexMax {
		return nil, util.ErrInvalidMethodParam
	}

	ownerTokenIDListKey := contractsAbi.GetTokenIDListKey(sendBlock.AccountAddress)
	oldIDList := util.GetValue(db, ownerTokenIDListKey)
	tokenInfo, _ := contractsAbi.ABIAsset.PackVariable(
		contractsAbi.VariableNameTokenInfo,
		params.Name,
		params.Symbol,
		params.TotalSupply,
		params.Decimals,
		sendBlock.AccountAddress,
		params.Mintable,
		params.MaxSupply,
		false,
		nextIndex)
	util.SetValue(db, key, tokenInfo)
	util.SetValue(db, ownerTokenIDListKey, contractsAbi.AppendTokenID(oldIDList, tokenID))
	b := big.NewInt(int64(nextIndex + 1)).FillBytes(make([]byte, 32))
	util.SetValue(db, nextIndexKey, b)

	event := NewLog(AbiAsset, "Issue", tokenID, sendBlock.AccountAddress)

	ret, err := methodAbi.Outputs.Pack(tokenID)

	if err != nil {
		return nil, err
	}

	// send the genesis transaction for the issued token
	genesisBlock := ledger.AccountBlock{
		BlockType: ledger.BlockTypeSendReward,
		TokenId: tokenID,
		Amount: params.TotalSupply,
		AccountAddress: sendBlock.ToAddress,
		ToAddress: sendBlock.AccountAddress,  // send to the issuer
	}

	return &NativeContractResult{
		Data: ret,
		TriggeredBlocks: []*ledger.AccountBlock{&genesisBlock},
		Events: []*ledger.VmLog{event},
	}, nil
}

func mint(request NativeContractRequest) (*NativeContractResult, error) {
	sendBlock := request.SendBlock
	methodAbi := request.Abi
	data := sendBlock.Data

	params := new(ParamMint)
	err := methodAbi.Inputs.Unpack(params, data[4:])
	if err != nil {
		return nil, err
	}
	// check params
	if params.Amount.Sign() <= 0 {
		return nil, util.ErrInvalidMethodParam
	}

	// mint
	var toAddress types.Address
	if params.To.IsZero() {
		toAddress = sendBlock.AccountAddress
	} else {
		toAddress = params.To
	}
	db := *request.Db
	oldTokenInfo, err := contractsAbi.GetTokenByID(db, params.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable ||
		new(big.Int).Sub(oldTokenInfo.MaxSupply, oldTokenInfo.TotalSupply).Cmp(params.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	// check owner
	if oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrAccessDenied
	}

	newTokenInfo, _ := contractsAbi.ABIAsset.PackVariable(
		contractsAbi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Add(oldTokenInfo.TotalSupply, params.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)
	util.SetValue(db, contractsAbi.GetTokenInfoKey(params.TokenId), newTokenInfo)

	event := NewLog(AbiAsset, "Mint", params.TokenId, toAddress, params.Amount)

	// send a mint transaction for the token
	mintBlock := ledger.AccountBlock{
		BlockType: ledger.BlockTypeSendReward,
		TokenId: params.TokenId,
		Amount: params.Amount,
		AccountAddress: sendBlock.ToAddress,
		ToAddress: toAddress,
		Data: []byte{},
	}
	mintBlock.Hash = mintBlock.ComputeHash()
	ret, err := methodAbi.Outputs.Pack(mintBlock.Hash.Bytes())

	return &NativeContractResult{
		Data: ret,
		TriggeredBlocks: []*ledger.AccountBlock{&mintBlock},
		Events: []*ledger.VmLog{event},
	}, err
}

func burn(request NativeContractRequest) (*NativeContractResult, error) {
	sendBlock := request.SendBlock

	// check params
	if bytes.Equal(sendBlock.TokenId.Bytes(), ledger.ViteTokenId.Bytes()) {
		// VITE can not be burnt by this method
		return nil, util.ErrInvalidMethodParam
	}
	if sendBlock.Amount.Sign() <= 0 {
		return nil, nil
	}
	db := *request.Db
	oldTokenInfo, err := contractsAbi.GetTokenByID(db, sendBlock.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable ||
		oldTokenInfo.TotalSupply.Cmp(sendBlock.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	// burn
	newTokenInfo, _ := contractsAbi.ABIAsset.PackVariable(
		contractsAbi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply.Sub(oldTokenInfo.TotalSupply, sendBlock.Amount),
		oldTokenInfo.Decimals,
		oldTokenInfo.Owner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)

	util.SetValue(db, contractsAbi.GetTokenInfoKey(sendBlock.TokenId), newTokenInfo)

	event := NewLog(AbiAsset, "Burn", sendBlock.TokenId, sendBlock.Amount)

	return &NativeContractResult{
		Data: PackedTrue,
		Events: []*ledger.VmLog{event},
	}, nil
}

func transferOwnership(request NativeContractRequest) (*NativeContractResult, error) {
	sendBlock := request.SendBlock
	methodAbi := request.Abi
	data := sendBlock.Data

	params := new(ParamTransferOwnership)
	err := methodAbi.Inputs.Unpack(params, data[4:])
	if err != nil {
		return nil, err
	}
	// check params
	if params.NewOwner == sendBlock.AccountAddress {
		return nil, util.ErrInvalidMethodParam
	}

	db := *request.Db
	oldTokenInfo, err := contractsAbi.GetTokenByID(db, params.TokenId)
	util.DealWithErr(err)
	if oldTokenInfo == nil || !oldTokenInfo.IsReIssuable {
		return nil, util.ErrInvalidMethodParam
	}
	if oldTokenInfo.Owner != sendBlock.AccountAddress {
		return nil, util.ErrAccessDenied
	}

	newTokenInfo, _ := contractsAbi.ABIAsset.PackVariable(
		contractsAbi.VariableNameTokenInfo,
		oldTokenInfo.TokenName,
		oldTokenInfo.TokenSymbol,
		oldTokenInfo.TotalSupply,
		oldTokenInfo.Decimals,
		params.NewOwner,
		oldTokenInfo.IsReIssuable,
		oldTokenInfo.MaxSupply,
		oldTokenInfo.OwnerBurnOnly,
		oldTokenInfo.Index)
	util.SetValue(db, contractsAbi.GetTokenInfoKey(params.TokenId), newTokenInfo)

	oldKey := contractsAbi.GetTokenIDListKey(sendBlock.AccountAddress)
	oldIDList := util.GetValue(db, oldKey)
	util.SetValue(db, oldKey, contractsAbi.DeleteTokenID(oldIDList, params.TokenId))
	newKey := contractsAbi.GetTokenIDListKey(params.NewOwner)
	newIDList := util.GetValue(db, newKey)
	util.SetValue(db, newKey, contractsAbi.AppendTokenID(newIDList, params.TokenId))

	event := NewLog(AbiAsset, "OwnershipTransferred", params.TokenId, oldTokenInfo.Owner, params.NewOwner)

	return &NativeContractResult{
		Data: PackedTrue,
		Events: []*ledger.VmLog{event},
	}, nil
}

func name(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.TokenName)

	return &NativeContractResult{Data: ret}, err
}

func symbol(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.TokenSymbol)

	return &NativeContractResult{Data: ret}, err
}

func decimals(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.Decimals)

	return &NativeContractResult{Data: ret}, err
}

func totalSupply(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.TotalSupply)

	return &NativeContractResult{Data: ret}, err
}

func maxSupply(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.MaxSupply)

	return &NativeContractResult{Data: ret}, err
}

func mintable(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.IsReIssuable)

	return &NativeContractResult{Data: ret}, err
}

func owner(request NativeContractRequest) (*NativeContractResult, error) {
	data := request.SendBlock.Data
	methodAbi := request.Abi
	tokenId := new(types.TokenTypeId)
	methodAbi.Inputs.Unpack(tokenId, data[4:])

	info, err := contractsAbi.GetTokenByID(*request.Db, *tokenId)
	if err != nil {
		return nil, err
	}
	ret, err := methodAbi.Outputs.Pack(info.Owner)

	return &NativeContractResult{Data: ret}, err
}