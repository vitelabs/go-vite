package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func HandleStakeAction(db vm_db.VmDb, stakeType uint8, actionType uint8, address types.Address, amount *big.Int, stakeHeight uint64) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if actionType == Stake {
		if methodData, err = stakeRequest(db, address, stakeType, amount, stakeHeight); err != nil {
			return []*ledger.AccountBlock{}, err
		} else {
			return []*ledger.AccountBlock{
				{
					AccountAddress: types.AddressDexFund,
					ToAddress:      types.AddressQuota,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         amount,
					TokenId:        ledger.ViteTokenId,
					Data:           methodData,
				},
			}, nil
		}
	} else {
		return DoCancelStake(db, address, stakeType, amount)
	}
}

func DoCancelStake(db vm_db.VmDb, address types.Address, stakeType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = cancelStakeRequest(db, address, stakeType, amount); err != nil {
		return []*ledger.AccountBlock{}, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressQuota,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           methodData,
			},
		}, nil
	}
}

func stakeRequest(db vm_db.VmDb, address types.Address, stakeType uint8, amount *big.Int, stakeHeight uint64) ([]byte, error) {
	if stakeType == StakeForVIP {
		if _, ok := GetVIPStaking(db, address); ok {
			return nil, VIPStakingExistsErr
		}
	} else if stakeType == StakeForSuperVIP {
		if _, ok := GetSuperVIPStaking(db, address); ok {
			return nil, SuperVipStakingExistsErr
		}
	}
	if _, err := ReduceAccount(db, address, ledger.ViteTokenId.Bytes(), amount); err != nil {
		return nil, err
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

func cancelStakeRequest(db vm_db.VmDb, address types.Address, stakeType uint8, amount *big.Int) ([]byte, error) {
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
		if _, ok := GetVIPStaking(db, address); !ok {
			return nil, VIPStakingNotExistsErr
		}
	case StakeForSuperVIP:
		if _, ok := GetSuperVIPStaking(db, address); !ok {
			return nil, SuperVIPStakingNotExistsErr
		}
	}
	var cancelStakeMethod = abi.MethodNameCancelDelegateStakeV2
	if !IsLeafFork(db) {
		cancelStakeMethod = abi.MethodNameCancelDelegateStake
	}
	if cancelStakeData, err := abi.ABIQuota.PackMethod(cancelStakeMethod, address, types.AddressDexFund, amount, uint8(stakeType)); err != nil {
		return nil, err
	} else {
		return cancelStakeData, err
	}
}

func OnMiningStakeSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) error {
	return doChangeMiningStakedAmount(db, reader, address, amount, updatedAmount)
}

func OnCancelMiningStakeSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) error {
	return doChangeMiningStakedAmount(db, reader, address, new(big.Int).Neg(amount), updatedAmount)
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
