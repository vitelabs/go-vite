package api

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"strings"
)

// Deprecated: use contract_createContractAddress instead
func (c *ContractApi) GetCreateContractToAddress(selfAddr types.Address, heightStr string, prevHash types.Hash) (*types.Address, error) {
	h, err := StringToUint64(heightStr)
	if err != nil {
		return nil, err
	}
	addr := util.NewContractAddress(selfAddr, h, prevHash)
	return &addr, nil
}

// Private
func (c *ContractApi) GetCreateContractParams(abiStr string, params []string) ([]byte, error) {
	if len(abiStr) > 0 && len(params) > 0 {
		abiContract, err := abi.JSONToABIContract(strings.NewReader(abiStr))
		if err != nil {
			return nil, err
		}
		arguments, err := convert(params, abiContract.Constructor.Inputs)
		if err != nil {
			return nil, err
		}
		constructorParams, err := abiContract.PackMethod("", arguments...)
		if err != nil {
			return nil, err
		}
		return constructorParams, nil
	}
	return []byte{}, nil
}

type CreateContractDataParam struct {
	Gid         types.Gid `json:"gid"`
	ConfirmTime uint8     `json:"confirmTime"`
	SeedCount   uint8     `json:"seedCount"`
	QuotaRatio  uint8     `json:"quotaRatio"`
	HexCode     string    `json:"hexCode"`
	Params      []byte    `json:"params"`
}

// Private
func (c *ContractApi) GetCreateContractData(param CreateContractDataParam) ([]byte, error) {
	code, err := hex.DecodeString(param.HexCode)
	if err != nil {
		return nil, err
	}
	if len(param.Params) > 0 {
		data := util.GetCreateContractData(
			helper.JoinBytes(code, param.Params),
			util.SolidityPPContractType,
			param.ConfirmTime,
			param.SeedCount,
			param.QuotaRatio,
			param.Gid)
		return data, nil
	} else {
		data := util.GetCreateContractData(
			code,
			util.SolidityPPContractType,
			param.ConfirmTime,
			param.SeedCount,
			param.QuotaRatio,
			param.Gid)
		return data, nil
	}
}

// Private
func (c *ContractApi) GetCallContractData(abiStr string, methodName string, params []string) ([]byte, error) {
	abiContract, err := abi.JSONToABIContract(strings.NewReader(abiStr))
	if err != nil {
		return nil, err
	}
	method, ok := abiContract.Methods[methodName]
	if !ok {
		return nil, errors.New("method name not found")
	}
	arguments, err := convert(params, method.Inputs)
	if err != nil {
		return nil, err
	}
	return abiContract.PackMethod(methodName, arguments...)
}

// Private
func (c *ContractApi) GetCallOffChainData(abiStr string, offChainName string, params []string) ([]byte, error) {
	abiContract, err := abi.JSONToABIContract(strings.NewReader(abiStr))
	if err != nil {
		return nil, err
	}
	method, ok := abiContract.OffChains[offChainName]
	if !ok {
		return nil, errors.New("offchain name not found")
	}
	arguments, err := convert(params, method.Inputs)
	if err != nil {
		return nil, err
	}
	return abiContract.PackOffChain(offChainName, arguments...)
}
