package chain_genesis

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"sort"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func NewGenesisAccountBlocks(cfg *config.Genesis) []*vm_db.VmAccountBlock {
	list := make([]*vm_db.VmAccountBlock, 0)
	addrSet := make(map[types.Address]interface{})
	list, addrSet = newGenesisGovernanceContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisAssetContractBlocks(cfg, list, addrSet)
	list, addrSet = newGenesisQuotaContractBlocks(cfg, list, addrSet)
	list = newGenesisNormalAccountBlocks(cfg, list, addrSet)
	return list
}

func updateAccountBalanceMap(cfg *config.Genesis, addr types.Address, vmdb vm_db.VmDb) {
	if len(cfg.AccountBalanceMap) == 0 {
		return
	}
	for tokenIdStr, balance := range cfg.AccountBalanceMap[addr.String()] {
		tokenId, err := types.HexToTokenTypeId(tokenIdStr)
		dealWithError(err)
		vmdb.SetBalance(&tokenId, balance)
	}
}

func newGenesisGovernanceContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.GovernanceInfo != nil {
		contractAddr := types.AddressGovernance
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		for gidStr, groupInfo := range cfg.GovernanceInfo.ConsensusGroupInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			var registerConditionParam []byte
			if groupInfo.RegisterConditionId == 1 {
				registerConditionParam, err = abi.ABIGovernance.PackVariable(abi.VariableNameRegisterStakeParam,
					groupInfo.RegisterConditionParam.StakeAmount,
					groupInfo.RegisterConditionParam.StakeToken,
					groupInfo.RegisterConditionParam.StakeHeight)
				dealWithError(err)
			}
			value, err := abi.ABIGovernance.PackVariable(abi.VariableNameConsensusGroupInfo,
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
				groupInfo.StakeAmount,
				groupInfo.ExpirationHeight)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetConsensusGroupInfoKey(gid), value)
		}

		for gidStr, groupRegistrationInfoMap := range cfg.GovernanceInfo.RegistrationInfoMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for name, registrationInfo := range groupRegistrationInfoMap {
				if len(registrationInfo.HistoryAddressList) == 0 {
					registrationInfo.HistoryAddressList = []types.Address{*registrationInfo.BlockProducingAddress}
				}
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegistrationInfo,
					name,
					registrationInfo.BlockProducingAddress,
					registrationInfo.StakeAddress,
					registrationInfo.Amount,
					registrationInfo.ExpirationHeight,
					registrationInfo.RewardTime,
					registrationInfo.RevokeTime,
					registrationInfo.HistoryAddressList)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetRegistrationInfoKey(name, gid), value)
				if len(cfg.GovernanceInfo.HisNameMap) == 0 ||
					len(cfg.GovernanceInfo.HisNameMap[gidStr]) == 0 ||
					len(cfg.GovernanceInfo.HisNameMap[gidStr][registrationInfo.BlockProducingAddress.String()]) == 0 {
					value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, name)
					dealWithError(err)
					util.SetValue(vmdb, abi.GetHisNameKey(*registrationInfo.BlockProducingAddress, gid), value)
				}
			}
		}

		for gidStr, groupHisNameMap := range cfg.GovernanceInfo.HisNameMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for blockProducingAddrStr, name := range groupHisNameMap {
				blockProducingAddr, err := types.HexToAddress(blockProducingAddrStr)
				dealWithError(err)
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, name)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetHisNameKey(blockProducingAddr, gid), value)
			}
		}

		for gidStr, groupVoteMap := range cfg.GovernanceInfo.VoteStatusMap {
			gid, err := types.HexToGid(gidStr)
			dealWithError(err)
			for voteAddrStr, sbpName := range groupVoteMap {
				voteAddr, err := types.HexToAddress(voteAddrStr)
				dealWithError(err)
				value, err := abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, sbpName)
				dealWithError(err)
				util.SetValue(vmdb, abi.GetVoteInfoKey(voteAddr, gid), value)
			}
		}

		updateAccountBalanceMap(cfg, contractAddr, vmdb)

		block.Hash = block.ComputeHash()
		list = append(list, &vm_db.VmAccountBlock{&block, vmdb})
		addrSet[contractAddr] = struct{}{}
	}
	return list, addrSet
}

type tokenInfoForSort struct {
	tokenId types.TokenTypeId
	config.TokenInfo
}
type byTokenId []*tokenInfoForSort

func (a byTokenId) Len() int      { return len(a) }
func (a byTokenId) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byTokenId) Less(i, j int) bool {
	return a[i].tokenId.Hex() > a[j].tokenId.Hex()
}

func newGenesisAssetContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.AssetInfo != nil {
		nextIndexMap := make(map[string]uint16)
		contractAddr := types.AddressAsset
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		tokenList := make([]*tokenInfoForSort, 0, len(cfg.AssetInfo.TokenInfoMap))
		for tokenIdStr, tokenInfo := range cfg.AssetInfo.TokenInfoMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			dealWithError(err)
			tokenList = append(tokenList, &tokenInfoForSort{tokenId, *tokenInfo})
		}
		sort.Sort(byTokenId(tokenList))
		for _, tokenInfo := range tokenList {
			nextIndex := uint16(0)
			if index, ok := nextIndexMap[tokenInfo.TokenSymbol]; ok {
				nextIndex = index
			}
			value, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenInfo,
				tokenInfo.TokenName,
				tokenInfo.TokenSymbol,
				tokenInfo.TotalSupply,
				tokenInfo.Decimals,
				tokenInfo.Owner,
				tokenInfo.IsReIssuable,
				tokenInfo.MaxSupply,
				tokenInfo.IsOwnerBurnOnly,
				nextIndex)
			dealWithError(err)
			nextIndex = nextIndex + 1
			nextIndexMap[tokenInfo.TokenSymbol] = nextIndex
			nextIndexValue, err := abi.ABIAsset.PackVariable(abi.VariableNameTokenIndex, nextIndex)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetNextTokenIndexKey(tokenInfo.TokenSymbol), nextIndexValue)
			util.SetValue(vmdb, abi.GetTokenInfoKey(tokenInfo.tokenId), value)
		}

		if len(cfg.AssetInfo.LogList) > 0 {
			for _, log := range cfg.AssetInfo.LogList {
				dataBytes, err := hex.DecodeString(log.Data)
				dealWithError(err)
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

func newGenesisQuotaContractBlocks(cfg *config.Genesis, list []*vm_db.VmAccountBlock, addrSet map[types.Address]interface{}) ([]*vm_db.VmAccountBlock, map[types.Address]interface{}) {
	if cfg.QuotaInfo != nil {
		contractAddr := types.AddressQuota
		block := ledger.AccountBlock{
			BlockType:      ledger.BlockTypeGenesisReceive,
			Height:         1,
			AccountAddress: contractAddr,
			Amount:         big.NewInt(0),
			Fee:            big.NewInt(0),
		}
		vmdb := vm_db.NewGenesisVmDB(&contractAddr)
		for stakeAddrStr, stakeInfoList := range cfg.QuotaInfo.StakeInfoMap {
			stakeAddr, err := types.HexToAddress(stakeAddrStr)
			dealWithError(err)
			for i, stakeInfo := range stakeInfoList {
				value, err := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo,
					stakeInfo.Amount,
					stakeInfo.ExpirationHeight,
					stakeInfo.Beneficiary,
					false,
					types.ZERO_ADDRESS,
					uint8(0))
				dealWithError(err)
				util.SetValue(vmdb, abi.GetStakeInfoKey(stakeAddr, uint64(i)), value)
			}
		}

		for beneficiaryStr, amount := range cfg.QuotaInfo.StakeBeneficialMap {
			beneficiary, err := types.HexToAddress(beneficiaryStr)
			dealWithError(err)
			value, err := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, amount)
			dealWithError(err)
			util.SetValue(vmdb, abi.GetStakeBeneficialKey(beneficiary), value)
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
		dealWithError(err)
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
		vmdb := vm_db.NewGenesisVmDB(&addr)
		for tokenIdStr, balance := range balanceMap {
			tokenId, err := types.HexToTokenTypeId(tokenIdStr)
			dealWithError(err)
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
