package client

import (
	"strings"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vm/abi"
)

type AbiClient interface {
	CallOffChain(methodName string, params ...interface{}) ([]interface{}, error)
}

type abiCli struct {
	cli          RpcClient
	contractAbi  *abi.ABIContract
	offchainCode string
	contractAddr types.Address
}

func GetAbiCli(cli RpcClient, abiCode string, offchainCode string, addr types.Address) (AbiClient, error) {
	contractAbi, err := abi.JSONToABIContract(strings.NewReader(abiCode))
	if err != nil {
		return nil, err
	}
	return &abiCli{cli: cli, contractAbi: &contractAbi, contractAddr: addr, offchainCode: offchainCode}, nil
}

func (cli abiCli) CallOffChain(methodName string, params ...interface{}) ([]interface{}, error) {
	data, err := cli.contractAbi.PackOffChain(methodName, params...)
	if err != nil {
		return nil, err
	}
	rpcParam := api.CallOffChainMethodParam{
		SelfAddr:     cli.contractAddr,
		OffChainCode: cli.offchainCode,
		Data:         data,
	}
	outputs, err := cli.cli.CallOffChainMethod(rpcParam)
	if err != nil {
		return nil, err
	}
	result, err := cli.contractAbi.DirectUnpackOffchainOutput(methodName, outputs)
	return result, err
}
