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

func HandleStackAction(db vm_db.VmDb, stackType uint8, actionType uint8, address types.Address, amount *big.Int, stakeHeight uint64) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if actionType == Stack {
		if methodData, err = stackRequest(db, address, stackType, amount, stakeHeight); err != nil {
			return []*ledger.AccountBlock{}, err
		} else {
			return []*ledger.AccountBlock{
				{
					AccountAddress: types.AddressDexFund,
					ToAddress:      types.AddressPledge,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         amount,
					TokenId:        ledger.ViteTokenId,
					Data:           methodData,
				},
			}, nil
		}
	} else {
		return DoCancelStack(db, address, stackType, amount)
	}
}

func DoCancelStack(db vm_db.VmDb, address types.Address, stackType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = cancelStackRequest(db, address, stackType, amount); err != nil {
		return []*ledger.AccountBlock{}, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressPledge,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           methodData,
			},
		}, nil
	}
}

func stackRequest(db vm_db.VmDb, address types.Address, stackType uint8, amount *big.Int, stakeHeight uint64) ([]byte, error) {
	if stackType == StackForVIP {
		if _, ok := GetStackedForVIP(db, address); ok {
			return nil, StackForVIPExistsErr
		}
	} else if stackType == StackForSuperVIP {
		if _, ok := GetStackedForSuperVIP(db, address); ok {
			return nil, StackForSuperVIPExistsErr
		}
	}
	if _, err := ReduceAccount(db, address, ledger.ViteTokenId.Bytes(), amount); err != nil {
		return nil, err
	} else {
		if stackData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, address, types.AddressDexFund, stackType, stakeHeight); err != nil {
			return nil, err
		} else {
			return stackData, err
		}
	}
}

func cancelStackRequest(db vm_db.VmDb, address types.Address, stackType uint8, amount *big.Int) ([]byte, error) {
	switch stackType {
	case StackForVx:
		available := GetStackedForVx(db, address)
		leave := new(big.Int).Sub(available, amount)
		if leave.Sign() < 0 {
			return nil, ExceedStackedAvailableErr
		} else if leave.Sign() > 0 && leave.Cmp(StackForVxMinAmount) < 0 {
			return nil, StackedAmountLeavedNotValidErr
		}
	case StackForVIP:
		if _, ok := GetStackedForVIP(db, address); !ok {
			return nil, StackedForVIPNotExistsErr
		}
	case StackForSuperVIP:
		if _, ok := GetStackedForSuperVIP(db, address); !ok {
			return nil, StackedForSuperVIPNotExistsErr
		}
	}
	if cancelStackData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, address, types.AddressDexFund, amount, uint8(stackType)); err != nil {
		return nil, err
	} else {
		return cancelStackData, err
	}
}

func OnStackForVxSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) error {
	return doChangeStackedForVxAmount(db, reader, address, amount, updatedAmount)
}

func OnCancelStackForVxSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) error {
	return doChangeStackedForVxAmount(db, reader, address, new(big.Int).Neg(amount), updatedAmount)
}

func doChangeStackedForVxAmount(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amtChange, updatedAmount *big.Int) error {
	var (
		stackedForVxs    *StackedForVxs
		sumChange        *big.Int
		periodId         uint64
		originStackedLen int
		needUpdate       bool
	)
	stackedForVxs, _ = GetStackedForVxs(db, address)
	periodId = GetCurrentPeriodId(db, reader)
	originStackedLen = len(stackedForVxs.Stacks)
	if originStackedLen == 0 { //need append new period
		if IsValidStackAmountForVx(updatedAmount) {
			stackedForVxByPeriod := &dexproto.StackedForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			stackedForVxs.Stacks = append(stackedForVxs.Stacks, stackedForVxByPeriod)
			sumChange = updatedAmount
			needUpdate = true
		}
	} else if stackedForVxs.Stacks[originStackedLen-1].Period == periodId { //update current period
		if IsValidStackAmountForVx(updatedAmount) {
			if IsValidStackAmountBytesForVx(stackedForVxs.Stacks[originStackedLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			stackedForVxs.Stacks[originStackedLen-1].Amount = updatedAmount.Bytes()
		} else {
			if IsValidStackAmountBytesForVx(stackedForVxs.Stacks[originStackedLen-1].Amount) {
				sumChange = NegativeAmount(stackedForVxs.Stacks[originStackedLen-1].Amount)
			}
			if originStackedLen > 1 { // in case originStackedLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				stackedForVxs.Stacks[originStackedLen-1].Amount = updatedAmount.Bytes()
			} else { // clear Stacks in case only current period saved and not valid any more
				stackedForVxs.Stacks = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidStackAmountForVx(updatedAmount) {
			if IsValidStackAmountBytesForVx(stackedForVxs.Stacks[originStackedLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			stackedForVxByPeriod := &dexproto.StackedForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			stackedForVxs.Stacks = append(stackedForVxs.Stacks, stackedForVxByPeriod)
			needUpdate = true
		} else {
			if IsValidStackAmountBytesForVx(stackedForVxs.Stacks[originStackedLen-1].Amount) {
				sumChange = NegativeAmount(stackedForVxs.Stacks[originStackedLen-1].Amount)
				stackedForVxByPeriod := &dexproto.StackedForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
				stackedForVxs.Stacks = append(stackedForVxs.Stacks, stackedForVxByPeriod)
				needUpdate = true
			}
		}
	}
	//update StacksSum
	if len(stackedForVxs.Stacks) > 0 && needUpdate {
		SaveStackedForVxs(db, address, stackedForVxs)
	} else if len(stackedForVxs.Stacks) == 0 && originStackedLen > 0 {
		DeleteStackedForVxs(db, address)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		dexStackedForVxs, _ := GetDexStackedForVxs(db)
		sumsLen := len(dexStackedForVxs.Stacks)
		if sumsLen == 0 {
			if sumChange.Sign() > 0 {
				dexStackedForVxs.Stacks = append(dexStackedForVxs.Stacks, &dexproto.StackedForVxByPeriod{Period: periodId, Amount: sumChange.Bytes()})
			} else {
				return fmt.Errorf("dexStackedForVxs initiation get negative value")
			}
		} else {
			sumRes := new(big.Int).Add(new(big.Int).SetBytes(dexStackedForVxs.Stacks[sumsLen-1].Amount), sumChange)
			if sumRes.Sign() < 0 {
				return fmt.Errorf("dexStackedForVxs updated res get negative value")
			}
			if dexStackedForVxs.Stacks[sumsLen-1].Period == periodId {
				dexStackedForVxs.Stacks[sumsLen-1].Amount = sumRes.Bytes()
			} else {
				dexStackedForVxs.Stacks = append(dexStackedForVxs.Stacks, &dexproto.StackedForVxByPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SaveDexStackedForVxs(db, dexStackedForVxs)
	}
	return nil
}
