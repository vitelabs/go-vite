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
	jsonQuota = `
	[
		{"type":"function","name":"Pledge", "inputs":[{"name":"beneficiary","type":"address"}]},
		{"type":"function","name":"Stake", "inputs":[{"name":"beneficiary","type":"address"}]},
		{"type":"function","name":"StakeForQuota", "inputs":[{"name":"beneficiary","type":"address"}]},

		{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"CancelStake","inputs":[{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"CancelQuotaStaking","inputs":[{"name":"id","type":"bytes32"}]},

		{"type":"function","name":"AgentPledge", "inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"bid","type":"uint8"},{"name":"stakeHeight","type":"uint64"}]},
		{"type":"function","name":"DelegateStake", "inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"bid","type":"uint8"},{"name":"stakeHeight","type":"uint64"}]},
		{"type":"function","name":"StakeForQuotaWithCallback", "inputs":[{"name":"beneficiary","type":"address"},{"name":"stakeHeight","type":"uint64"}]},	

		{"type":"function","name":"AgentCancelPledge","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"}]},
		{"type":"function","name":"CancelDelegateStake","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"}]},
		{"type":"function","name":"CancelQuotaStakingWithCallback","inputs":[{"name":"id","type":"bytes32"}]},

		{"type":"callback","name":"AgentPledge","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"callback","name":"DelegateStake","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"callback","name":"StakeForQuotaWithCallback", "inputs":[{"name":"id","type":"bytes32"},{"name":"success","type":"bool"}]},	

		{"type":"callback","name":"AgentCancelPledge","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"callback","name":"CancelDelegateStake","inputs":[{"name":"stakeAddress","type":"address"},{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
		{"type":"callback","name":"CancelQuotaStakingWithCallback","inputs":[{"name":"id","type":"bytes32"},{"name":"success","type":"bool"}]},

		{"type":"variable","name":"stakeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"expirationHeight","type":"uint64"},{"name":"beneficiary","type":"address"},{"name":"isDelegated","type":"bool"},{"name":"delegateAddress","type":"address"},{"name":"bid","type":"uint8"}]},

		{"type":"variable","name":"stakeInfoV2","inputs":[{"name":"amount","type":"uint256"},{"name":"expirationHeight","type":"uint64"},{"name":"beneficiary","type":"address"},{"name":"id","type":"bytes32"}]},

		{"type":"variable","name":"stakeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNameStake                   = "Pledge"
	MethodNameStakeV2                 = "Stake"
	MethodNameStakeV3                 = "StakeForQuota"
	MethodNameCancelStake             = "CancelPledge"
	MethodNameCancelStakeV2           = "CancelStake"
	MethodNameCancelStakeV3           = "CancelQuotaStaking"
	MethodNameDelegateStake           = "AgentPledge"
	MethodNameDelegateStakeV2         = "DelegateStake"
	MethodNameCancelDelegateStake     = "AgentCancelPledge"
	MethodNameCancelDelegateStakeV2   = "CancelDelegateStake"
	MethodNameStakeWithCallback       = "StakeForQuotaWithCallback"
	MethodNameCancelStakeWithCallback = "CancelQuotaStakingWithCallback"
	VariableNameStakeInfo             = "stakeInfo"
	VariableNameStakeInfoV2           = "stakeInfoV2"
	VariableNameStakeBeneficial       = "stakeBeneficial"
)

var (
	// ABIQuota is abi definition of quota contract
	ABIQuota, _        = abi.JSONToABIContract(strings.NewReader(jsonQuota))
	stakeInfoKeySize   = types.AddressSize + 8
	stakeInfoValueSize = 192
)

// VariableStakeBeneficial defines variable of stake beneficial amount in quota contract
type VariableStakeBeneficial struct {
	Amount *big.Int
}

// ParamCancelStake defines parameters of cancel stake method in quota contract
type ParamCancelStake struct {
	Beneficiary types.Address
	Amount      *big.Int
}

// ParamDelegateStake defines parameters of delegate stake method in quota contract
type ParamDelegateStake struct {
	StakeAddress types.Address
	Beneficiary  types.Address
	Bid          uint8
	StakeHeight  uint64
}

type ParamStakeV3 struct {
	StakeAddress types.Address
	Beneficiary  types.Address
	StakeHeight  uint64
}

// ParamCancelDelegateStake defines parameters of cancel delegate stake method in quota contract
type ParamCancelDelegateStake struct {
	StakeAddress types.Address
	Beneficiary  types.Address
	Amount       *big.Int
	Bid          uint8
}

// GetStakeBeneficialKey generate db key for stake beneficial amount
func GetStakeBeneficialKey(beneficial types.Address) []byte {
	return beneficial.Bytes()
}

// GetStakeInfoKey generate db key for stake info
func GetStakeInfoKey(addr types.Address, index uint64) []byte {
	return append(addr.Bytes(), helper.LeftPadBytes(new(big.Int).SetUint64(index).Bytes(), 8)...)
}

// GetStakeInfoKeyPrefix is used for db iterator
func GetStakeInfoKeyPrefix(addr types.Address) []byte {
	return addr.Bytes()
}

// IsStakeInfoKey check whether a db key is stake info key
func IsStakeInfoKey(key []byte) bool {
	return len(key) == stakeInfoKeySize
}

// GetStakeAddrFromStakeInfoKey decode stake address from a stake info key
func GetStakeAddrFromStakeInfoKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[:types.AddressSize])
	return address
}

// GetIndexFromStakeInfoKey decode index from a stake info key
func GetIndexFromStakeInfoKey(key []byte) uint64 {
	return new(big.Int).SetBytes(key[types.AddressSize:]).Uint64()
}

// GetStakeInfoList query stake info list by stake address, excluding delegate stake
func GetStakeInfoList(db StorageDatabase, stakeAddr types.Address) ([]*types.StakeInfo, *big.Int, error) {
	if *db.Address() != types.AddressQuota {
		return nil, nil, util.ErrAddressNotMatch
	}
	stakeAmount := big.NewInt(0)
	iterator, err := db.NewStorageIterator(GetStakeInfoKeyPrefix(stakeAddr))
	if err != nil {
		return nil, nil, err
	}
	defer iterator.Release()
	stakeInfoList := make([]*types.StakeInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsStakeInfoKey) {
			continue
		}
		if stakeInfo, err := UnpackStakeInfo(iterator.Value()); err == nil {
			if stakeInfo.Amount != nil && stakeInfo.Amount.Sign() > 0 && !stakeInfo.IsDelegated {
				stakeInfoList = append(stakeInfoList, stakeInfo)
				if stakeInfo.Amount != nil && stakeInfo.Amount.Sign() > 0 {
					stakeAmount.Add(stakeAmount, stakeInfo.Amount)
				}
			}
		}
	}
	return stakeInfoList, stakeAmount, nil
}

func GetStakeExpirationHeight(db StorageDatabase, list []*types.StakeInfo) ([]*types.StakeInfo, error) {
	if *db.Address() != types.AddressQuota {
		return nil, util.ErrAddressNotMatch
	}
	for _, s := range list {
		var stakeInfo *types.StakeInfo
		var err error
		if s.Id == nil {
			stakeInfo, err = GetStakeInfo(db, s.StakeAddress, s.Beneficiary, s.DelegateAddress, s.IsDelegated, s.Bid)
		} else {
			stakeInfoKey, err := db.GetValue(s.Id.Bytes())
			if err != nil {
				return nil, err
			}
			if len(stakeInfoKey) == 0 {
				return nil, util.ErrChainForked
			}
			stakeInfo, err = GetStakeInfoByKey(db, stakeInfoKey)
		}
		if err != nil {
			return nil, err
		} else if stakeInfo == nil {
			return nil, util.ErrChainForked
		} else {
			s.ExpirationHeight = stakeInfo.ExpirationHeight
		}
	}
	return list, nil
}

// GetStakeBeneficialAmount query stake amount of beneficiary
func GetStakeBeneficialAmount(db StorageDatabase, beneficiary types.Address) (*big.Int, error) {
	if *db.Address() != types.AddressQuota {
		return nil, util.ErrAddressNotMatch
	}
	v, err := db.GetValue(GetStakeBeneficialKey(beneficiary))
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return big.NewInt(0), nil
	}
	amount := new(VariableStakeBeneficial)
	ABIQuota.UnpackVariable(amount, VariableNameStakeBeneficial, v)
	return amount.Amount, nil
}

// GetStakeInfo query exact stake info by stake address, delegate address, delegate type and bid
func GetStakeInfo(db StorageDatabase, stakeAddr, beneficiary, delegateAddr types.Address, isDelegated bool, bid uint8) (*types.StakeInfo, error) {
	iterator, err := db.NewStorageIterator(GetStakeInfoKeyPrefix(stakeAddr))
	util.DealWithErr(err)
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !IsStakeInfoKey(iterator.Key()) {
			continue
		}
		stakeInfo, _ := UnpackStakeInfo(iterator.Value())
		if stakeInfo.Beneficiary == beneficiary && stakeInfo.IsDelegated == isDelegated &&
			stakeInfo.DelegateAddress == delegateAddr && stakeInfo.Bid == bid {
			return stakeInfo, nil
		}
	}
	return nil, nil
}

func GetStakeInfoByKey(db StorageDatabase, stakeInfoKey []byte) (*types.StakeInfo, error) {
	value, err := db.GetValue(stakeInfoKey)
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, nil
	}
	return UnpackStakeInfo(value)
}

func GetStakeInfoById(db StorageDatabase, id []byte) (*types.StakeInfo, error) {
	if len(id) != types.HashSize {
		return nil, util.ErrInvalidMethodParam
	}
	if storeKey, err := db.GetValue(id); err != nil {
		return nil, err
	} else {
		if value, err := db.GetValue(storeKey); err != nil {
			return nil, err
		} else if len(value) > 0 {
			return UnpackStakeInfo(value)
		}
	}
	return nil, util.ErrDataNotExist
}

func UnpackStakeInfo(value []byte) (*types.StakeInfo, error) {
	stakeInfo := new(types.StakeInfo)
	if len(value) == stakeInfoValueSize {
		if err := ABIQuota.UnpackVariable(stakeInfo, VariableNameStakeInfo, value); err != nil {
			return stakeInfo, err
		}
	} else {
		stakeInfo.Id = &types.Hash{}
		if err := ABIQuota.UnpackVariable(stakeInfo, VariableNameStakeInfoV2, value); err != nil {
			return stakeInfo, err
		}
	}
	return stakeInfo, nil
}

// GetStakeListByPage batch query stake info under certain snapshot block status continuously by last query key
func GetStakeListByPage(db StorageDatabase, lastKey []byte, count uint64) ([]*types.StakeInfo, []byte, error) {
	iterator, err := db.NewStorageIterator(nil)
	util.DealWithErr(err)
	defer iterator.Release()

	if len(lastKey) > 0 {
		ok := iterator.Seek(lastKey)
		if !ok {
			return nil, nil, nil
		}
	}
	stakeInfoList := make([]*types.StakeInfo, 0, count)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, iterator.Error()
			}
			break
		}
		if !IsStakeInfoKey(iterator.Key()) {
			continue
		}
		stakeInfo, err := UnpackStakeInfo(iterator.Value())
		if err != nil {
			continue
		}
		stakeInfo.StakeAddress = GetStakeAddrFromStakeInfoKey(iterator.Key())
		stakeInfoList = append(stakeInfoList, stakeInfo)
		count = count - 1
		if count == 0 {
			break
		}
	}
	return stakeInfoList, iterator.Key(), nil
}
