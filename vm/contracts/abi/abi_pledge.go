package abi

import (
	"github.com/vitelabs/go-vite/common/helper"
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
		{"type":"function","name":"AgentPledge", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"bid","type":"uint8"},{"name":"stakeHeight","type":"uint64"}]},
		{"type":"function","name":"AgentCancelPledge","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"}]},
		{"type":"callback","name":"AgentPledge","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"callback","name":"AgentCancelPledge","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"beneficialAddr","type":"address"},{"name":"agent","type":"bool"},{"name":"agentAddress","type":"address"},{"name":"bid","type":"uint8"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNamePledge             = "Pledge"
	MethodNameCancelPledge       = "CancelPledge"
	MethodNameAgentPledge        = "AgentPledge"
	MethodNameAgentCancelPledge  = "AgentCancelPledge"
	VariableNamePledgeInfo       = "pledgeInfo"
	VariableNamePledgeBeneficial = "pledgeBeneficial"
)

var (
	ABIPledge, _  = abi.JSONToABIContract(strings.NewReader(jsonPledge))
	pledgeKeySize = types.AddressSize + 8
)

type VariablePledgeBeneficial struct {
	Amount *big.Int
}
type ParamCancelPledge struct {
	Beneficial types.Address
	Amount     *big.Int
}
type ParamAgentPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Bid           uint8
	StakeHeight   uint64
}
type ParamAgentCancelPledge struct {
	PledgeAddress types.Address
	Beneficial    types.Address
	Amount        *big.Int
	Bid           uint8
}

func GetPledgeBeneficialKey(beneficial types.Address) []byte {
	return beneficial.Bytes()
}
func GetPledgeKey(addr types.Address, index uint64) []byte {
	return append(addr.Bytes(), helper.LeftPadBytes(new(big.Int).SetUint64(index).Bytes(), 8)...)
}
func GetPledgeKeyPrefix(addr types.Address) []byte {
	return addr.Bytes()
}
func IsPledgeKey(key []byte) bool {
	return len(key) == pledgeKeySize
}

func GetPledgeAddrFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[:types.AddressSize])
	return address
}
func GetIndexFromPledgeKey(key []byte) uint64 {
	return new(big.Int).SetBytes(key[types.AddressSize:]).Uint64()
}

func GetPledgeInfoList(db StorageDatabase, pledgeAddr types.Address) ([]*types.PledgeInfo, *big.Int, error) {
	if *db.Address() != types.AddressPledge {
		return nil, nil, util.ErrAddressNotMatch
	}
	pledgeAmount := big.NewInt(0)
	iterator, err := db.NewStorageIterator(GetPledgeKeyPrefix(pledgeAddr))
	if err != nil {
		return nil, nil, err
	}
	defer iterator.Release()
	pledgeInfoList := make([]*types.PledgeInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsPledgeKey) {
			continue
		}
		pledgeInfo := new(types.PledgeInfo)
		if err := ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, iterator.Value()); err == nil &&
			pledgeInfo.Amount != nil && pledgeInfo.Amount.Sign() > 0 {
			pledgeInfoList = append(pledgeInfoList, pledgeInfo)
			pledgeAmount.Add(pledgeAmount, pledgeInfo.Amount)
		}
	}
	return pledgeInfoList, pledgeAmount, nil
}

func GetPledgeBeneficialAmount(db StorageDatabase, beneficialAddr types.Address) (*big.Int, error) {
	if *db.Address() != types.AddressPledge {
		return nil, util.ErrAddressNotMatch
	}
	v, err := db.GetValue(GetPledgeBeneficialKey(beneficialAddr))
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return big.NewInt(0), nil
	}
	amount := new(VariablePledgeBeneficial)
	ABIPledge.UnpackVariable(amount, VariableNamePledgeBeneficial, v)
	return amount.Amount, nil
}

func GetPledgeInfo(db StorageDatabase, pledgeAddr, beneficialAddr, agentAddr types.Address, agent bool, bid uint8) (*types.PledgeInfo, error) {
	iterator, err := db.NewStorageIterator(GetPledgeKeyPrefix(pledgeAddr))
	util.DealWithErr(err)
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !IsPledgeKey(iterator.Key()) {
			continue
		}
		pledgeInfo := new(types.PledgeInfo)
		ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, iterator.Value())
		if pledgeInfo.BeneficialAddr == beneficialAddr && pledgeInfo.Agent == agent &&
			pledgeInfo.AgentAddress == agentAddr && pledgeInfo.Bid == bid {
			return pledgeInfo, nil
		}
	}
	return nil, nil
}

func GetPledgeListByPage(db StorageDatabase, lastKey []byte, count uint64) ([]*types.PledgeInfo, []byte, error) {
	iterator, err := db.NewStorageIterator(nil)
	util.DealWithErr(err)
	defer iterator.Release()

	if len(lastKey) > 0 {
		ok := iterator.Seek(lastKey)
		if !ok {
			return nil, nil, nil
		}
	}
	pledgeInfoList := make([]*types.PledgeInfo, 0, count)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, iterator.Error()
			}
			break
		}
		if !IsPledgeKey(iterator.Key()) {
			continue
		}
		pledgeInfo := new(types.PledgeInfo)
		if err := ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, iterator.Value()); err != nil {
			continue
		}
		pledgeInfo.PledgeAddress = GetPledgeAddrFromPledgeKey(iterator.Key())
		pledgeInfoList = append(pledgeInfoList, pledgeInfo)
		count = count - 1
		if count == 0 {
			break
		}
	}
	return pledgeInfoList, iterator.Key(), nil
}
