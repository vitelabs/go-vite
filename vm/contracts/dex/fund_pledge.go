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

func HandlePledgeAction(db vm_db.VmDb, pledgeType uint8, actionType uint8, address types.Address, amount *big.Int, stakeHeight uint64) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if actionType == Pledge {
		if methodData, err = pledgeRequest(db, address, pledgeType, amount, stakeHeight); err != nil {
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
		return DoCancelPledge(db, address, pledgeType, amount)
	}
}

func DoCancelPledge(db vm_db.VmDb, address types.Address, pledgeType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = cancelPledgeRequest(db, address, pledgeType, amount); err != nil {
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

func pledgeRequest(db vm_db.VmDb, address types.Address, pledgeType uint8, amount *big.Int, stakeHeight uint64) ([]byte, error) {
	if pledgeType == PledgeForVip {
		if _, ok := GetPledgeForVip(db, address); ok {
			return nil, PledgeForVipExistsErr
		}
	}
	if _, err := SubUserFund(db, address, ledger.ViteTokenId.Bytes(), amount); err != nil {
		return nil, err
	} else {
		if pledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, address, types.AddressDexFund, pledgeType, stakeHeight); err != nil {
			return nil, err
		} else {
			return pledgeData, err
		}
	}
}

func cancelPledgeRequest(db vm_db.VmDb, address types.Address, pledgeType uint8, amount *big.Int) ([]byte, error) {
	if pledgeType == PledgeForVx {
		available := GetPledgeForVx(db, address)
		leave := new(big.Int).Sub(available, amount)
		if leave.Sign() < 0 {
			return nil, ExceedPledgeAvailableErr
		} else if leave.Sign() > 0 && leave.Cmp(PledgeForVxMinAmount) < 0 {
			return nil, PledgeAmountLeavedNotValidErr
		}
	} else {
		if _, ok := GetPledgeForVip(db, address); !ok {
			return nil, PledgeForVipNotExistsErr
		}
	}
	if cancelPledgeData, err := abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, address, types.AddressDexFund, amount, uint8(pledgeType)); err != nil {
		return nil, err
	} else {
		return cancelPledgeData, err
	}
}

func OnPledgeForVxSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) {
	doChangePledgedVxAmount(db, reader, address, amount, updatedAmount)
}

func OnCancelPledgeForVxSuccess(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount, updatedAmount *big.Int) {
	doChangePledgedVxAmount(db, reader, address, new(big.Int).Neg(amount), updatedAmount)
}

func doChangePledgedVxAmount(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amtChange, updatedAmount *big.Int) {
	var (
		pledges          *PledgesForVx
		sumChange        *big.Int
		periodId         uint64
		originPledgesLen int
		needUpdate       bool
	)
	pledges, _ = GetPledgesForVx(db, address)
	periodId = GetCurrentPeriodId(db, reader)
	originPledgesLen = len(pledges.Pledges)
	if originPledgesLen == 0 { //need append new period
		if IsValidPledgeAmountForVx(updatedAmount) {
			pledgeForVxByPeriod := &dexproto.PledgeForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			pledges.Pledges = append(pledges.Pledges, pledgeForVxByPeriod)
			sumChange = updatedAmount
			needUpdate = true
		}
	} else if pledges.Pledges[originPledgesLen-1].Period == periodId { //update current period
		if IsValidPledgeAmountForVx(updatedAmount) {
			if IsValidPledgeAmountBytesForVx(pledges.Pledges[originPledgesLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			pledges.Pledges[originPledgesLen-1].Amount = updatedAmount.Bytes()
		} else {
			if IsValidPledgeAmountBytesForVx(pledges.Pledges[originPledgesLen-1].Amount) {
				sumChange = NegativeAmount(pledges.Pledges[originPledgesLen-1].Amount)
			}
			if originPledgesLen > 1 { // in case originPledgesLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				pledges.Pledges[originPledgesLen-1].Amount = updatedAmount.Bytes()
			} else { // clear Pledges in case only current period saved and not valid any more
				pledges.Pledges = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidPledgeAmountForVx(updatedAmount) {
			if IsValidPledgeAmountBytesForVx(pledges.Pledges[originPledgesLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = updatedAmount
			}
			pledgeForVxByPeriod := &dexproto.PledgeForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
			pledges.Pledges = append(pledges.Pledges, pledgeForVxByPeriod)
			needUpdate = true
		} else {
			if IsValidPledgeAmountBytesForVx(pledges.Pledges[originPledgesLen-1].Amount) {
				sumChange = NegativeAmount(pledges.Pledges[originPledgesLen-1].Amount)
				pledgeForVxByPeriod := &dexproto.PledgeForVxByPeriod{Period: periodId, Amount: updatedAmount.Bytes()}
				pledges.Pledges = append(pledges.Pledges, pledgeForVxByPeriod)
				needUpdate = true
			}
		}
	}
	//update pledgesSum
	if len(pledges.Pledges) > 0 && needUpdate {
		SavePledgesForVx(db, address, pledges)
	} else if len(pledges.Pledges) == 0 && originPledgesLen > 0 {
		DeletePledgesForVx(db, address)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		pledgesForVxSum, _ := GetPledgesForVxSum(db)
		sumsLen := len(pledgesForVxSum.Pledges)
		if sumsLen == 0 {
			if sumChange.Sign() > 0 {
				pledgesForVxSum.Pledges = append(pledgesForVxSum.Pledges, &dexproto.PledgeForVxByPeriod{Period: periodId, Amount: sumChange.Bytes()})
			} else {
				panic(fmt.Errorf("vxPledgesum initiation get negative value"))
			}
		} else {
			sumRes := new(big.Int).Add(new(big.Int).SetBytes(pledgesForVxSum.Pledges[sumsLen-1].Amount), sumChange)
			if sumRes.Sign() < 0 {
				panic(fmt.Errorf("vxPledgesum updated res get negative value"))
			}
			if pledgesForVxSum.Pledges[sumsLen-1].Period == periodId {
				pledgesForVxSum.Pledges[sumsLen-1].Amount = sumRes.Bytes()
			} else {
				pledgesForVxSum.Pledges = append(pledgesForVxSum.Pledges, &dexproto.PledgeForVxByPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SavePledgesForVxSum(db, pledgesForVxSum)
	}
}
