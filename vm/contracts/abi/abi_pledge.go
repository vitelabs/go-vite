package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strings"
)

const (
	jsonPledge = `
	[
		{"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNamePledge             = "Pledge"
	MethodNameCancelPledge       = "CancelPledge"
	VariableNamePledgeInfo       = "pledgeInfo"
	VariableNamePledgeBeneficial = "pledgeBeneficial"
)

var (
	ABIPledge, _ = abi.JSONToABIContract(strings.NewReader(jsonPledge))
)

type VariablePledgeBeneficial struct {
	Amount *big.Int
}
type ParamCancelPledge struct {
	Beneficial types.Address
	Amount     *big.Int
}
type PledgeInfo struct {
	Amount         *big.Int
	WithdrawHeight uint64
	BeneficialAddr types.Address
}

func GetPledgeBeneficialKey(beneficial types.Address) []byte {
	return beneficial.Bytes()
}
func GetPledgeKey(addr types.Address, beneficial types.Address) []byte {
	return append(addr.Bytes(), beneficial.Bytes()...)
}
func IsPledgeKey(key []byte) bool {
	return len(key) == 2*types.AddressSize
}
func GetBeneficialFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[types.AddressSize:])
	return address
}

func GetPledgeAddrFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[:types.AddressSize])
	return address
}

func GetPledgeInfoList(db StorageDatabase, addr types.Address) ([]*PledgeInfo, *big.Int, error) {
	if *db.Address() != types.AddressPledge {
		return nil, nil, util.ErrAddressNotMatch
	}
	pledgeAmount := big.NewInt(0)
	iterator, err := db.NewStorageIterator(nil)
	if err != nil {
		return nil, nil, err
	}
	defer iterator.Release()
	pledgeInfoList := make([]*PledgeInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsPledgeKey) {
			continue
		}
		pledgeInfo := new(PledgeInfo)
		if err := ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, iterator.Value()); err == nil && pledgeInfo.Amount != nil && pledgeInfo.Amount.Sign() > 0 {
			pledgeInfo.BeneficialAddr = GetBeneficialFromPledgeKey(iterator.Key())
			pledgeInfoList = append(pledgeInfoList, pledgeInfo)
			pledgeAmount.Add(pledgeAmount, pledgeInfo.Amount)
		}
	}
	return pledgeInfoList, pledgeAmount, nil
}
