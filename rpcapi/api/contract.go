package api

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
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

func (c *ContractApi) GetCreateContractToAddress(selfAddr types.Address, height uint64, prevHash types.Hash, snapshotHash types.Hash) types.Address {
	return util.NewContractAddress(selfAddr, height, prevHash, snapshotHash)
}

func (c *ContractApi) GetCreateContractData(gid types.Gid, hexCode string) ([]byte, error) {
	code, err := hex.DecodeString(hexCode)
	if err != nil {
		return nil, err
	}
	return append(gid.Bytes(), code...), nil
}

func (c *ContractApi) GetCallContractData(abiStr string, methodName string, params []string) (*string, error) {
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
	result, err := abiContract.PackMethod(methodName, arguments...)
	if err != nil {
		return nil, err
	}
	resultStr := hex.EncodeToString(result)
	return &resultStr, err
}
