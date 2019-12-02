package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"sort"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetCandidateList(sd, gid)
}

func (c *chain) GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetAllRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetAllRegistrationList(sd, gid)
}

func (c *chain) GetConsensusGroup(snapshotHash types.Hash, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetConsensusGroup")
		return nil, cErr
	}

	return abi.GetConsensusGroup(sd, gid)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetConsensusGroupList")
		return nil, cErr
	}

	// do something
	return abi.GetConsensusGroupList(sd)
}

func (c *chain) GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetVoteList")
		return nil, cErr
	}

	// do something
	return abi.GetVoteList(sd, gid)
}

func (c *chain) GetStakeBeneficialAmount(addr types.Address) (*big.Int, error) {

	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressQuota)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetStakeBeneficialAmount")
		return nil, cErr
	}

	// do something
	return abi.GetStakeBeneficialAmount(sd, addr)
}

// total
func (c *chain) GetStakeQuota(addr types.Address) (*big.Int, *types.Quota, error) {

	amount, err := c.GetStakeBeneficialAmount(addr)
	if err != nil {
		return nil, nil, err
	}

	vmDb := vm_db.NewVmDbByAddr(c, &addr)

	quota, err := quota.GetQuota(vmDb, addr, amount, c.GetLatestSnapshotBlock().Height)
	if err != nil {
		return nil, nil, err
	}
	return amount, &quota, nil
}

// total
func (c *chain) GetStakeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error) {
	sb := c.GetLatestSnapshotBlock()
	sd, err := c.stateDB.NewStorageDatabase(sb.Hash, types.AddressQuota)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetStakeBeneficialAmount")
		return nil, cErr
	}

	quotaMap := make(map[types.Address]*types.Quota, len(addrList))

	for _, addr := range addrList {
		amount, err := abi.GetStakeBeneficialAmount(sd, addr)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("abi.GetStakeBeneficialAmount failed, Addr is %s. Error: %s", addr, err))
			c.log.Error(cErr.Error(), "method", "GetStakeQuotas")
			return nil, err
		}

		vmDb := vm_db.NewVmDbByAddr(c, &addr)
		quota, err := quota.GetQuota(vmDb, addr, amount, sb.Height)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("quota.GetQuota failed. Error: %s", err))
			c.log.Error(cErr.Error(), "method", "GetStakeQuotas")
			return nil, err
		}
		quotaMap[addr] = &quota
	}
	return quotaMap, nil

}

func (c *chain) GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressAsset)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetTokenInfoById")

		return nil, cErr
	}
	return abi.GetTokenByID(sd, tokenId)
}

func (c *chain) GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressAsset)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetAllTokenInfo")

		return nil, cErr
	}
	return abi.GetTokenMap(sd)
}

type ByBalance []*interfaces.VoteDetails

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[j].Balance.Cmp(a[i].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	}
	return r < 0
}

func (c *chain) CalVoteDetails(gid types.Gid, info *core.GroupInfo, snapshotBlock ledger.HashHeight) ([]*interfaces.VoteDetails, error) {
	// query register info
	registerList, _ := c.GetRegisterList(snapshotBlock.Hash, gid)
	// query vote info
	votes, _ := c.GetVoteList(snapshotBlock.Hash, gid)

	var registers []*interfaces.VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, c.genVoteDetails(snapshotBlock.Hash, v, votes, info.CountingTokenId))
	}
	sort.Sort(ByBalance(registers))
	return registers, nil
}

func (c *chain) genVoteDetails(snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *interfaces.VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.SbpName == registration.Name {
			addrs = append(addrs, v.VoteAddr)
		}
	}
	balanceMap, _ := c.GetConfirmedBalanceList(addrs, id, snapshotHash)
	balanceTotal := big.NewInt(0)
	for _, v := range balanceMap {
		balanceTotal.Add(balanceTotal, v)
	}
	return &interfaces.VoteDetails{
		Vote: core.Vote{
			Name:    registration.Name,
			Addr:    registration.BlockProducingAddress,
			Balance: balanceTotal,
		},
		CurrentAddr:  registration.BlockProducingAddress,
		RegisterList: registration.HisAddrList,
		Addr:         balanceMap,
	}
}

func (c *chain) GetStakeListByPage(snapshotHash types.Hash, lastKey []byte, count uint64) ([]*types.StakeInfo, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressQuota)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetStakeAmountByPage")

		return nil, nil, cErr
	}
	return abi.GetStakeListByPage(sd, lastKey, count)
}

func (c *chain) GetDexFundsByPage(snapshotHash types.Hash, lastAddress types.Address, count int) ([]*dex.Fund, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetStakeAmountByPage")
		return nil, cErr
	}
	return dex.GetUserFundsByPage(sd, lastAddress, count)
}

func (c *chain) GetDexStakeListByPage(snapshotHash types.Hash, lastKey []byte, count int) ([]*dex.DelegateStakeInfo, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetDexStakeListByPage")
		return nil, nil, cErr
	}
	return dex.GetStakeListByPage(sd, lastKey, count)
}
