package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

type builtinContract struct {
	m   map[string]contracts.BuiltinContractMethod
	abi abi.ABIContract
}

var simpleContracts = map[types.Address]*builtinContract{
	types.AddressPledge: {
		map[string]contracts.BuiltinContractMethod{
			cabi.MethodNamePledge:       &contracts.MethodPledge{},
			cabi.MethodNameCancelPledge: &contracts.MethodCancelPledge{},
		},
		cabi.ABIPledge,
	},
	types.AddressConsensusGroup: {
		map[string]contracts.BuiltinContractMethod{
			/*contracts.MethodNameCreateConsensusGroup:   &contracts.MethodCreateConsensusGroup{},
			contracts.MethodNameCancelConsensusGroup:   &contracts.MethodCancelConsensusGroup{},
			contracts.MethodNameReCreateConsensusGroup: &contracts.MethodReCreateConsensusGroup{},*/
			cabi.MethodNameRegister:           &contracts.MethodRegister{},
			cabi.MethodNameCancelRegister:     &contracts.MethodCancelRegister{},
			cabi.MethodNameReward:             &contracts.MethodReward{},
			cabi.MethodNameUpdateRegistration: &contracts.MethodUpdateRegistration{},
			cabi.MethodNameVote:               &contracts.MethodVote{},
			cabi.MethodNameCancelVote:         &contracts.MethodCancelVote{},
		},
		cabi.ABIConsensusGroup,
	},
	types.AddressMintage: {
		map[string]contracts.BuiltinContractMethod{
			cabi.MethodNameCancelMintPledge: &contracts.MethodMintageCancelPledge{},
			cabi.MethodNameMint:             &contracts.MethodMint{},
			cabi.MethodNameIssue:            &contracts.MethodIssue{},
			cabi.MethodNameBurn:             &contracts.MethodBurn{},
			cabi.MethodNameTransferOwner:    &contracts.MethodTransferOwner{},
			cabi.MethodNameChangeTokenType:  &contracts.MethodChangeTokenType{},
		},
		cabi.ABIMintage,
	},
}

func GetBuiltinContract(addr types.Address, methodSelector []byte) (contracts.BuiltinContractMethod, bool, error) {
	p, ok := simpleContracts[addr]
	if ok {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, ok := p.m[method.Name]
			return c, ok, nil
		} else {
			return nil, ok, util.ErrAbiMethodNotFound
		}
	}
	return nil, ok, nil
}
