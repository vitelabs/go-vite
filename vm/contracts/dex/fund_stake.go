package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)
//TODO refactory for simple
func HandleStakeAction(db vm_db.VmDb, stakeType, actionType uint8, address, principal types.Address, amount *big.Int, stakeHeight uint64, block *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	if actionType == Stake {
		if methodData, err := stakeRequest(db, address, principal, stakeType, amount, stakeHeight); err != nil {
			return []*ledger.AccountBlock{}, err
		} else {
			blocks := []*ledger.AccountBlock{
				{
					AccountAddress: types.AddressDexFund,
					ToAddress:      types.AddressQuota,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         amount,
					TokenId:        ledger.ViteTokenId,
					Data:           methodData,
				},
			}
			if IsEarthFork(db) {
				stakeId := util.ComputeSendBlockHash(block, blocks[0], 0)
				SaveDelegateStakeInfo(db, stakeId, stakeType, address, principal, amount)
			}
			return blocks, nil
		}
	} else {
		return DoCancelStake(db, address, principal, stakeType, amount)
	}
}

// |              | withoutId | withId |
// | Mining       |    Yes    |   No   |
// | VIP          |    Yes    |   Yes  |
// | SVIP         |    Yes    |   Yes  |
// | PrincipalSVIP|     -     |   Yes  |
func DoCancelStake(db vm_db.VmDb, address, principal types.Address, stakeType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = cancelStakeRequest(db, address, principal, stakeType, amount); err != nil {
		return []*ledger.AccountBlock{}, err
	} else {
		return composeCancelBlock(methodData), nil
	}
}

func DoCancelStakeWithId(id types.Hash)  ([]*ledger.AccountBlock, error) {
	if methodData, err := abi.ABIQuota.PackMethod(abi.MethodNameCancelStakeWithCallback, id); err != nil {
		return []*ledger.AccountBlock{}, err
	} else {
		return composeCancelBlock(methodData), nil
	}
}

func stakeRequest(db vm_db.VmDb, address, principal types.Address, stakeType uint8, amount *big.Int, stakeHeight uint64) ([]byte, error) {
	switch stakeType {
	case StakeForVIP:
		if _, ok := GetVIPStaking(db, address); ok {
			return nil, VIPStakingExistsErr
		}
	case StakeForSuperVIP:
		if _, ok := GetSuperVIPStaking(db, address); ok {
			return nil, SuperVipStakingExistsErr
		}
	case StakeForPrincipalSuperVIP:
		if _, ok := GetSuperVIPStaking(db, principal); ok {
			return nil, SuperVipStakingExistsErr
		}
	}
	if _, err := ReduceAccount(db, address, ledger.ViteTokenId.Bytes(), amount); err != nil {
		return nil, err
	} else {
		if IsEarthFork(db) {
			if stakeData, err := abi.ABIQuota.PackMethod(abi.MethodNameStakeWithCallback, types.AddressDexFund, stakeHeight); err != nil {
				return nil, err
			} else {
				return stakeData, err
			}
		} else {
			var stakeMethod = abi.MethodNameDelegateStakeV2
			if !IsLeafFork(db) {
				stakeMethod = abi.MethodNameDelegateStake
			}
			if stakeData, err := abi.ABIQuota.PackMethod(stakeMethod, address, types.AddressDexFund, stakeType, stakeHeight); err != nil {
				return nil, err
			} else {
				return stakeData, err
			}
		}
	}
}

func cancelStakeRequest(db vm_db.VmDb, address, principal types.Address, stakeType uint8, amount *big.Int) ([]byte, error) {
	var (
		vipStaking *VIPStaking
		stakeId    []byte
		ok         bool
	)
	switch stakeType {
	case StakeForMining:
		available := GetMiningStakedAmount(db, address)
		leave := new(big.Int).Sub(available, amount)
		if leave.Sign() < 0 {
			return nil, ExceedStakedAvailableErr
		} else if leave.Sign() > 0 && leave.Cmp(StakeForMiningMinAmount) < 0 {
			return nil, StakingAmountLeavedNotValidErr
		}
	case StakeForVIP:
		if vipStaking, ok = GetVIPStaking(db, address); !ok {
			return nil, VIPStakingNotExistsErr
		}
	case StakeForSuperVIP:
		if vipStaking, ok = GetSuperVIPStaking(db, address); !ok {
			return nil, SuperVIPStakingNotExistsErr
		}
	case StakeForPrincipalSuperVIP:
		if vipStaking, ok = GetSuperVIPStaking(db, principal); !ok {
			return nil, SuperVIPStakingNotExistsErr
		}
	}
	if stakeType == StakeForMining || !IsVipStakingWithId(vipStaking) { // cancel old version stake
		var cancelStakeMethod = abi.MethodNameCancelDelegateStakeV2
		if !IsLeafFork(db) {
			cancelStakeMethod = abi.MethodNameCancelDelegateStake
		}
		return abi.ABIQuota.PackMethod(cancelStakeMethod, address, types.AddressDexFund, amount, uint8(stakeType))
	} else if IsVipStakingWithId(vipStaking) {
		var matchStakeAddr bool
		for _, id := range vipStaking.StakingHashes {
			if info, ok := GetDelegateStakeInfo(db, id); ok {
				if bytes.Equal(info.Address, address.Bytes()) {
					stakeId = id
					matchStakeAddr = true
					break
				}
			}
		}
		if !matchStakeAddr {
			return nil, OnlyOwnerAllowErr
		}
		return abi.ABIQuota.PackMethod(abi.MethodNameCancelStakeWithCallback, stakeId)
	} else {
		return nil, InvalidOperationErr
	}
}


func composeCancelBlock(methodData []byte) []*ledger.AccountBlock {
	return []*ledger.AccountBlock{
		{
			AccountAddress: types.AddressDexFund,
			ToAddress:      types.AddressQuota,
			BlockType:      ledger.BlockTypeSendCall,
			TokenId:        ledger.ViteTokenId,
			Amount:         big.NewInt(0),
			Data:           methodData,
		},
	}
}

func IsVipStakingWithId(staking *VIPStaking) bool {
	if len(staking.StakingHashes) > 0 {
		if int(staking.StakedTimes) == len(staking.StakingHashes) {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func GetStakeInfoList(db vm_db.VmDb, stakeAddr types.Address, filter func(*DelegateStakeAddressIndex)bool) ([]*DelegateStakeInfo, *big.Int, error) {
	if *db.Address() != types.AddressDexFund {
		return nil, nil, InvalidInputParamErr
	}
	stakeAmount := big.NewInt(0)
	iterator, err := db.NewStorageIterator(append(delegateStakeAddressIndexPrefix, stakeAddr.Bytes()...))
	if err != nil {
		return nil, nil, err
	}
	defer iterator.Release()
	stakeInfoList := make([]*DelegateStakeInfo, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, nil, iterator.Error()
			}
			break
		}
		stakeIndex := &DelegateStakeAddressIndex{}
		if ok := deserializeFromDb(db, iterator.Key(), stakeIndex); ok {
			if filter(stakeIndex) {
				if info, ok := GetDelegateStakeInfo(db, stakeIndex.Id); ok {
					info.Id = stakeIndex.Id
					stakeInfoList = append(stakeInfoList, info)
					stakeAmount.Add(stakeAmount, new(big.Int).SetBytes(info.Amount))
				}
			}
		}
	}
	return stakeInfoList, stakeAmount, nil
}

func OnMiningStakeSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount, updatedV2Amount *big.Int) error {
	return doChangeMiningStakedAmount(db, reader, address, amount, new(big.Int).Add(updatedAmount, updatedV2Amount))
}

func OnCancelMiningStakeSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount, updatedV2Amount *big.Int) error {
	return doChangeMiningStakedAmount(db, reader, address, new(big.Int).Neg(amount), new(big.Int).Add(updatedAmount, updatedV2Amount))
}

func doChangeMiningStakedAmount(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amtChange, updatedAmount *big.Int) error {
	var (
		miningStakings    *MiningStakings
		sumChange         *big.Int
		periodId          uint64
		originStakingsLen int
		needUpdate        bool
	)
	miningStakings, _ = GetMiningStakings(db, address)
	periodId = GetCurrentPeriodId(db, reader)
	originStakingsLen = len(miningStakings.Stakings)
	if originStakingsLen == 0 { //need append new period
		if IsValidMiningStakeAmount(updatedAmount) {
			miningStakingByPeriod := &dexproto.MiningStakingByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			miningStakings.Stakings = append(miningStakings.Stakings, miningStakingByPeriod)
			sumChange = updatedAmount
			needUpdate = true
		}
	} else if miningStakings.Stakings[originStakingsLen-1].Period == periodId { //update current period
		if IsValidMiningStakeAmount(updatedAmount) {
			if IsValidMiningStakeAmountBytes(miningStakings.Stakings[originStakingsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			miningStakings.Stakings[originStakingsLen-1].Amount = updatedAmount.Bytes()
		} else {
			if IsValidMiningStakeAmountBytes(miningStakings.Stakings[originStakingsLen-1].Amount) {
				sumChange = NegativeAmount(miningStakings.Stakings[originStakingsLen-1].Amount)
			}
			if originStakingsLen > 1 { // in case originStakingsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				miningStakings.Stakings[originStakingsLen-1].Amount = updatedAmount.Bytes()
			} else { // clear Stakings in case only current period saved and not valid any more
				miningStakings.Stakings = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidMiningStakeAmount(updatedAmount) {
			if IsValidMiningStakeAmountBytes(miningStakings.Stakings[originStakingsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			miningStakingByPeriod := &dexproto.MiningStakingByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			miningStakings.Stakings = append(miningStakings.Stakings, miningStakingByPeriod)
			needUpdate = true
		} else {
			if IsValidMiningStakeAmountBytes(miningStakings.Stakings[originStakingsLen-1].Amount) {
				sumChange = NegativeAmount(miningStakings.Stakings[originStakingsLen-1].Amount)
				miningStakingByPeriod := &dexproto.MiningStakingByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
				miningStakings.Stakings = append(miningStakings.Stakings, miningStakingByPeriod)
				needUpdate = true
			}
		}
	}
	//update MiningStakings
	if len(miningStakings.Stakings) > 0 && needUpdate {
		SaveMiningStakings(db, address, miningStakings)
	} else if len(miningStakings.Stakings) == 0 && originStakingsLen > 0 {
		DeleteMiningStakings(db, address)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		dexMiningStakings, _ := GetDexMiningStakings(db)
		dexStakingsLen := len(dexMiningStakings.Stakings)
		if dexStakingsLen == 0 {
			if sumChange.Sign() > 0 {
				dexMiningStakings.Stakings = append(dexMiningStakings.Stakings, &dexproto.MiningStakingByPeriod{Period: periodId, Amount: sumChange.Bytes()})
			} else {
				return fmt.Errorf("dexMiningStakings initiation get negative value")
			}
		} else {
			sumRes := new(big.Int).Add(new(big.Int).SetBytes(dexMiningStakings.Stakings[dexStakingsLen-1].Amount), sumChange)
			if sumRes.Sign() < 0 {
				return fmt.Errorf("dexMiningStakings updated res get negative value")
			}
			if dexMiningStakings.Stakings[dexStakingsLen-1].Period == periodId {
				dexMiningStakings.Stakings[dexStakingsLen-1].Amount = sumRes.Bytes()
			} else {
				dexMiningStakings.Stakings = append(dexMiningStakings.Stakings, &dexproto.MiningStakingByPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SaveDexMiningStakings(db, dexMiningStakings)
	}
	return nil
}