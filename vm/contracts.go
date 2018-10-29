package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts"
)

type precompiledContract struct {
	m   map[string]contracts.PrecompiledContractMethod
	abi abi.ABIContract
}

var simpleContracts = map[types.Address]*precompiledContract{
	contracts.AddressRegister: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNameRegister:           &contracts.MethodRegister{},
			contracts.MethodNameCancelRegister:     &contracts.MethodCancelRegister{},
			contracts.MethodNameReward:             &contracts.MethodReward{},
			contracts.MethodNameUpdateRegistration: &contracts.MethodUpdateRegistration{},
		},
		contracts.ABIRegister,
	},
	contracts.AddressVote: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNameVote:       &contracts.MethodVote{},
			contracts.MethodNameCancelVote: &contracts.MethodCancelVote{},
		},
		contracts.ABIVote,
	},
	contracts.AddressPledge: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNamePledge:       &contracts.MethodPledge{},
			contracts.MethodNameCancelPledge: &contracts.MethodCancelPledge{},
		},
		contracts.ABIPledge,
	},
	contracts.AddressConsensusGroup: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNameCreateConsensusGroup:   &contracts.MethodCreateConsensusGroup{},
			contracts.MethodNameCancelConsensusGroup:   &contracts.MethodCancelConsensusGroup{},
			contracts.MethodNameReCreateConsensusGroup: &contracts.MethodReCreateConsensusGroup{},
		},
		contracts.ABIConsensusGroup,
	},
	contracts.AddressMintage: {
		map[string]contracts.PrecompiledContractMethod{
			contracts.MethodNameMintage:             &contracts.MethodMintage{},
			contracts.MethodNameMintageCancelPledge: &contracts.MethodMintageCancelPledge{},
		},
		contracts.ABIMintage,
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
	return nil, false, nil
}
