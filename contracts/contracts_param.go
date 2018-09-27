package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/abi"
	"errors"
)

// pack params to byte slice
// unpack byte silce to param variables
// pack consensus group condition params by condition id
// unpack consensus group condition param byte slice by condition id

var (
	precompiledContractsAbiMap = map[types.Address]abi.ABIContract {
		AddressRegister: ABIRegister,
		AddressVote: ABIVote,
		AddressPledge: ABIPledge,
		AddressConsensusGroup: ABIConsensusGroup,
		AddressMintage: ABIMintage,
	}
	errInvalidParam = errors.New("invalid param")
)

// register
func PackMethodParam(contractsAddr types.Address, methodName string, params ...interface{}) ([]byte, error) {
	if abiContract, ok := precompiledContractsAbiMap[contractsAddr]; ok {
		if data, err := abiContract.PackMethod(methodName, params...); err == nil {
			return data, nil
		}
	}
	return nil, errInvalidParam
}


