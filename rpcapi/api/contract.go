package api

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"strings"
)

type ContractApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewContractApi(vite *vite.Vite) *ContractApi {
	return &ContractApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/contract_api"),
	}
}

func (c ContractApi) String() string {
	return "ContractApi"
}

func (c *ContractApi) GetCreateContractToAddress(selfAddr types.Address, heightStr string, prevHash types.Hash) (*types.Address, error) {
	h, err := StringToUint64(heightStr)
	if err != nil {
		return nil, err
	}
	addr := util.NewContractAddress(selfAddr, h, prevHash)
	return &addr, nil
}

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

func (c *ContractApi) GetCreateContractData(param CreateContractDataParam) ([]byte, error) {
	code, err := hex.DecodeString(param.HexCode)
	if err != nil {
		return nil, err
	}
	if !util.IsValidQuotaRatio(param.QuotaRatio) {
		return nil, util.ErrInvalidQuotaRatio
	}
	sb := c.chain.GetLatestSnapshotBlock()
	if len(param.Params) > 0 {
		data := util.GetCreateContractData(
			helper.JoinBytes(code, param.Params),
			util.SolidityPPContractType,
			param.ConfirmTime,
			param.SeedCount,
			param.QuotaRatio,
			param.Gid,
			sb.Height)
		return data, nil
	} else {
		data := util.GetCreateContractData(
			code,
			util.SolidityPPContractType,
			param.ConfirmTime,
			param.SeedCount,
			param.QuotaRatio,
			param.Gid,
			sb.Height)
		return data, nil
	}
}

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

type CallOffChainMethodParam struct {
	SelfAddr          types.Address `json:"selfAddr"`
	OffChainCode      string        `json:"offchainCode"`
	OffChainCodeBytes []byte        `json:"offchainCodeBytes"`
	Data              []byte        `json:"data"`
}

func (c *ContractApi) CallOffChainMethod(param CallOffChainMethodParam) ([]byte, error) {
	prevHash, err := getPrevBlockHash(c.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(c.chain, &param.SelfAddr, &c.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	var codeBytes []byte
	if len(param.OffChainCode) > 0 {
		codeBytes, err = hex.DecodeString(param.OffChainCode)
		if err != nil {
			return nil, err
		}
	} else {
		codeBytes = param.OffChainCodeBytes
	}
	return vm.NewVM(nil).OffChainReader(db, codeBytes, param.Data)
}

func (c *ContractApi) GetContractStorage(addr types.Address, prefix string) (map[string]string, error) {
	var prefixBytes []byte
	if len(prefix) > 0 {
		var err error
		prefixBytes, err = hex.DecodeString(prefix)
		if err != nil {
			return nil, err
		}
	}
	iter, err := c.chain.GetStorageIterator(addr, prefixBytes)
	if err != nil {
		return nil, err
	}
	defer iter.Release()
	m := make(map[string]string)
	for {
		if !iter.Next() {
			if iter.Error() != nil {
				return nil, iter.Error()
			}
			return m, nil
		}
		if len(iter.Key()) > 0 && len(iter.Value()) > 0 {
			m["0x"+hex.EncodeToString(iter.Key())] = "0x" + hex.EncodeToString(iter.Value())
		}
	}
}

type ContractInfo struct {
	Code        []byte    `json:"code"`
	Gid         types.Gid `json:"gid"`
	ConfirmTime uint8     `json:"confirmTime"`
	SeedCount   uint8     `json:"seedCount"`
	QuotaRatio  uint8     `json:"quotaRatio"`
}

func (c *ContractApi) GetContractInfo(addr types.Address) (*ContractInfo, error) {
	code, err := c.chain.GetContractCode(addr)
	if err != nil {
		return nil, err
	}
	meta, err := c.chain.GetContractMeta(addr)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}
	return &ContractInfo{Code: code, Gid: meta.Gid, ConfirmTime: meta.SendConfirmedTimes, SeedCount: meta.SeedConfirmedTimes, QuotaRatio: meta.QuotaRatio}, nil
}
