package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
)

type precompiledContract struct {
	m   map[string]contracts.PrecompiledContractMethod
	abi abi.ABIContract
}

var simpleContracts = map[types.Address]*precompiledContract{
	cabi.AddressRegister: {
		map[string]contracts.PrecompiledContractMethod{
			cabi.MethodNameRegister:       &contracts.MethodRegister{},
			cabi.MethodNameCancelRegister: &contracts.MethodCancelRegister{},
			// TODO not support reward this version cabi.MethodNameReward:             &contracts.MethodReward{},
			cabi.MethodNameUpdateRegistration: &contracts.MethodUpdateRegistration{},
		},
		cabi.ABIRegister,
	},
	cabi.AddressVote: {
		map[string]contracts.PrecompiledContractMethod{
			cabi.MethodNameVote:       &contracts.MethodVote{},
			cabi.MethodNameCancelVote: &contracts.MethodCancelVote{},
		},
		cabi.ABIVote,
	},
	cabi.AddressPledge: {
		map[string]contracts.PrecompiledContractMethod{
			cabi.MethodNamePledge:       &contracts.MethodPledge{},
			cabi.MethodNameCancelPledge: &contracts.MethodCancelPledge{},
		},
		cabi.ABIPledge,
	},
	/* TODO not support consensus group this version
	contracts.AddressConsensusGroup: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNameCreateConsensusGroup:   &contracts.MethodCreateConsensusGroup{},
			contracts.MethodNameCancelConsensusGroup:   &contracts.MethodCancelConsensusGroup{},
			contracts.MethodNameReCreateConsensusGroup: &contracts.MethodReCreateConsensusGroup{},
		},
		contracts.ABIConsensusGroup,
	},*/
	cabi.AddressMintage: {
		map[string]contracts.PrecompiledContractMethod{
			cabi.MethodNameMintage:             &contracts.MethodMintage{},
			cabi.MethodNameMintageCancelPledge: &contracts.MethodMintageCancelPledge{},
		},
		cabi.ABIMintage,
	},
}

func isPrecompiledContractAddress(addr types.Address) bool {
	_, ok := simpleContracts[addr]
	return ok
}
func getPrecompiledContract(addr types.Address, methodSelector []byte) (contracts.PrecompiledContractMethod, bool, error) {
	p, ok := simpleContracts[addr]
	if ok {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, ok := p.m[method.Name]
			return c, ok, nil
		} else {
			return nil, ok, err
		}
	}
	return nil, ok, nil
}
