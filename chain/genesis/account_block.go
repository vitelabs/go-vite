package chain_genesis

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func NewGenesisAccountBlocks(cfg *config.Genesis) []*vm_db.VmAccountBlock {
	list := make([]*vm_db.VmAccountBlock, 0)
	addrSet := make(map[types.Address]interface{})
	list, addrSet = newGenesisConsensusGroupContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisMintageContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisPledgeContractBlocks(cfg, list, addrSet)
	list = newGenesisNormalAccountBlocks(cfg, list, addrSet)
	return list
}

func updateAccountBalanceMap(cfg *config.Genesis, addr types.Address, vmdb vm_db.VmDb) {
	if len(cfg.AccountBalanceMap) == 0 {
		return
	}
	for tokenIdStr, balance := range cfg.AccountBalanceMap[addr.String()] {
		tokenId, err := types.HexToTokenTypeId(tokenIdStr)
		if err != nil {
			panic(err)
		}
		vmdb.SetBalance(&tokenId, balance)
	}
}

func newGenesisConsensusGroupContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.ConsensusGroupInfo != nil {
		contractAddr := types.AddressConsensusGroup
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewEmptyVmDB(&contractAddr)
		for gidStr, groupInfo := range cfg.ConsensusGroupInfo.ConsensusGroupInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			var registerConditionParam []byte
			if groupInfo.RegisterConditionId == 1 {
				registerConditionParam, err = abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge,
					groupInfo.RegisterConditionParam.PledgeAmount,
					groupInfo.RegisterConditionParam.PledgeToken,
					groupInfo.RegisterConditionParam.PledgeHeight)
				dealWithError(err)
			}
			value, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
				groupInfo.NodeCount,
				groupInfo.Interval,
				groupInfo.PerCount,
				groupInfo.RandCount,
				groupInfo.RandRank,
				groupInfo.Repeat,
				groupInfo.CheckLevel,
				groupInfo.CountingTokenId,
				groupInfo.RegisterConditionId,
				registerConditionParam,
				groupInfo.VoteConditionId,
				[]byte{},
				groupInfo.Owner,
				groupInfo.PledgeAmount,
				groupInfo.WithdrawHeight)
			dealWithError(err)
			err = vmdb.SetValue(abi.GetConsensusGroupKey(gid), value)
			dealWithError(err)
		}

		for gidStr, groupRegistrationInfoMap := range cfg.ConsensusGroupInfo.RegistrationInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for name, registrationInfo := range groupRegistrationInfoMap {
				if len(registrationInfo.HisAddrList) == 0 {
					registrationInfo.HisAddrList = []types.Address{registrationInfo.NodeAddr}
				}
				value, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration,
					name,
					registrationInfo.NodeAddr,
					registrationInfo.PledgeAddr,
					registrationInfo.Amount,
					registrationInfo.WithdrawHeight,
					registrationInfo.RewardTime,
					registrationInfo.CancelTime,
					registrationInfo.HisAddrList)
				dealWithError(err)
				err = vmdb.SetValue(abi.GetRegisterKey(name, gid), value)
				dealWithError(err)
				if len(cfg.ConsensusGroupInfo.HisNameMap) == 0 ||
					len(cfg.ConsensusGroupInfo.HisNameMap[gidStr]) == 0 ||
					len(cfg.ConsensusGroupInfo.HisNameMap[gidStr][registrationInfo.NodeAddr.String()]) == 0 {
					value, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, name)
					dealWithError(err)
					err = vmdb.SetValue(abi.GetHisNameKey(registrationInfo.NodeAddr, gid), value)
					dealWithError(err)
				}
			}
		}

		for gidStr, groupHisNameMap := range cfg.ConsensusGroupInfo.HisNameMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for nodeAddrStr, name := range groupHisNameMap {
				nodeAddr, err := types.HexToAddress(nodeAddrStr)
				dealWithError(err)
				value, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, name)
				dealWithError(err)
				err = vmdb.SetValue(abi.GetHisNameKey(nodeAddr, gid), value)
				dealWithError(err)
			}
		}

		for gidStr, groupVoteMap := range cfg.ConsensusGroupInfo.VoteStatusMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for voteAddrStr, nodeName := range groupVoteMap {
				voteAddr, err := types.HexToAddress(voteAddrStr)
				dealWithError(err)
				value, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameVoteStatus, nodeName)
				dealWithError(err)
				err = vmdb.SetValue(abi.GetVoteKey(voteAddr, gid), value)
				dealWithError(err)
			}
		}

		updateAccountBalanceMap(cfg, contractAddr, vmdb)

		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newGenesisMintageContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.MintageInfo != nil {
		nextIndexMap := make(map[string]uint16)
		contractAddr := types.AddressMintage
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewEmptyVmDB(&contractAddr)
		for tokenIdStr, tokenInfo := range cfg.MintageInfo.TokenInfoMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			dealWithError(err)
			nextIndex := uint16(0)
			if index, ok := nextIndexMap[tokenInfo.TokenSymbol]; ok {
				nextIndex = index
			}
			value, err := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo,
				tokenInfo.TokenName,
				tokenInfo.TokenSymbol,
				tokenInfo.TotalSupply,
				tokenInfo.Decimals,
				tokenInfo.Owner,
				tokenInfo.PledgeAmount,
				tokenInfo.WithdrawHeight,
				tokenInfo.PledgeAddr,
				tokenInfo.IsReIssuable,
				tokenInfo.MaxSupply,
				tokenInfo.OwnerBurnOnly,
				nextIndex)
			dealWithError(err)
			nextIndex = nextIndex + 1
			nextIndexMap[tokenInfo.TokenSymbol] = nextIndex
			nextIndexValue, err := abi.ABIMintage.PackVariable(abi.VariableNameTokenNameIndex, nextIndex)
			dealWithError(err)
			err = vmdb.SetValue(abi.GetNextIndexKey(tokenInfo.TokenSymbol), nextIndexValue)
			dealWithError(err)
			err = vmdb.SetValue(abi.GetMintageKey(tokenId), value)
			dealWithError(err)
		}

		if len(cfg.MintageInfo.LogList) > 0 {
			for _, log := range cfg.MintageInfo.LogList {
				dataBytes, err := hex.DecodeString(log.Data)
				if err != nil {
					panic(err)
				}
				vmdb.AddLog(&ledger.VmLog{Data: dataBytes, Topics: log.Topics})
			}
		}
		block.LogHash = vmdb.GetLogListHash()
		updateAccountBalanceMap(cfg, contractAddr, vmdb)
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newGenesisPledgeContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.PledgeInfo != nil {
		contractAddr := types.AddressPledge
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewEmptyVmDB(&contractAddr)
		for pledgeAddrStr, pledgeInfoList := range cfg.PledgeInfo.PledgeInfoMap {
			pledgeAddr, err := types.HexToAddress(pledgeAddrStr)
			dealWithError(err)
			for i, pledgeInfo := range pledgeInfoList {
				value, err := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo,
					pledgeInfo.Amount,
					pledgeInfo.WithdrawHeight,
					pledgeInfo.BeneficialAddr)
				dealWithError(err)
				err = vmdb.SetValue(abi.GetPledgeKey(pledgeAddr, uint64(i)), value)
				dealWithError(err)
			}
		}

		for beneficialAddrStr, amount := range cfg.PledgeInfo.PledgeBeneficialMap {
			beneficialAddr, err := types.HexToAddress(beneficialAddrStr)
			dealWithError(err)
			value, err := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, amount)
			dealWithError(err)
			err = vmdb.SetValue(abi.GetPledgeBeneficialKey(beneficialAddr), value)
			dealWithError(err)
		}
		updateAccountBalanceMap(cfg, contractAddr, vmdb)
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

func newGenesisNormalAccountBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) []*vm_db.VmAccountBlock {
	for addrStr, balanceMap := range cfg.AccountBalanceMap {
		addr, err := types.HexToAddress(addrStr)
		if err != nil {
			panic(err)
		}
		if _, ok := addrSet[addr]; ok {
			continue
		}
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: addr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewEmptyVmDB(&addr)
		for tokenIdStr, balance := range balanceMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			if err != nil {
				panic(err)
			}
			vmdb.SetBalance(&tokenId, balance)
		}
		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
	}

	return list
}

func dealWithError(err error) {
	if err != nil {
		panic(err)
	}
}
