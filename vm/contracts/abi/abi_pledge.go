package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
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
func GetPledgeKey(addr types.Address, pledgeBeneficialKey []byte) []byte {
	return append(addr.Bytes(), pledgeBeneficialKey...)
}
func IsPledgeKey(key []byte) bool {
	return len(key) == 2*types.AddressSize
}
func GetBeneficialFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[types.AddressSize:])
	return address
}

func GetPledgeBeneficialAmount(db StorageDatabase, beneficial types.Address) *big.Int {
	key := GetPledgeBeneficialKey(beneficial)
	beneficialAmount := new(VariablePledgeBeneficial)
	if err := ABIPledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.GetStorageBySnapshotHash(&AddressPledge, key, nil)); err == nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

func GetPledgeInfoList(db StorageDatabase, addr types.Address) ([]*PledgeInfo, *big.Int) {
	pledgeAmount := big.NewInt(0)
	iterator := db.NewStorageIteratorBySnapshotHash(&AddressPledge, addr.Bytes(), nil)
	pledgeInfoList := make([]*PledgeInfo, 0)
	if iterator == nil {
		return pledgeInfoList, pledgeAmount
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsPledgeKey(key) {
			pledgeInfo := new(PledgeInfo)
			if err := ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, value); err == nil && pledgeInfo.Amount != nil && pledgeInfo.Amount.Sign() > 0 {
				pledgeInfo.BeneficialAddr = GetBeneficialFromPledgeKey(key)
				pledgeInfoList = append(pledgeInfoList, pledgeInfo)
				pledgeAmount.Add(pledgeAmount, pledgeInfo.Amount)
			}
		}
	}
	return pledgeInfoList, pledgeAmount
}
