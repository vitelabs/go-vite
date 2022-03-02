package chain

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/vm/contracts/abi"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
	"github.com/vitelabs/go-vite/v2/vm/quota"
	"github.com/vitelabs/go-vite/v2/vm_db"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash)
		c.log.Error(cErr.Error(), "method", "GetRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetCandidateList(sd, gid)
}

func (c *chain) GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash)
		c.log.Error(cErr.Error(), "method", "GetAllRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetAllRegistrationList(sd, gid)
}

func (c *chain) GetConsensusGroup(snapshotHash types.Hash, gid types.Gid) (*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash)
		c.log.Error(cErr.Error(), "method", "GetConsensusGroup")
		return nil, cErr
	}

	return abi.GetConsensusGroup(sd, gid)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash)
		c.log.Error(cErr.Error(), "method", "GetConsensusGroupList")
		return nil, cErr
	}

	// do something
	return abi.GetConsensusGroupList(sd)
}

func (c *chain) GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressGovernance)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash)
		c.log.Error(cErr.Error(), "method", "GetVoteList")
		return nil, cErr
	}

	// do something
	return abi.GetVoteList(sd, gid)
}

func (c *chain) GetStakeBeneficialAmount(addr types.Address) (*big.Int, error) {

	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressQuota)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
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
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetStakeBeneficialAmount")
		return nil, cErr
	}

	quotaMap := make(map[types.Address]*types.Quota, len(addrList))

	for _, addr := range addrList {
		amount, err := abi.GetStakeBeneficialAmount(sd, addr)
		if err != nil {
			cErr := fmt.Errorf("abi.GetStakeBeneficialAmount failed, Addr is %s. Error: %s", addr, err)
			c.log.Error(cErr.Error(), "method", "GetStakeQuotas")
			return nil, err
		}

		vmDb := vm_db.NewVmDbByAddr(c, &addr)
		quota, err := quota.GetQuota(vmDb, addr, amount, sb.Height)
		if err != nil {
			cErr := fmt.Errorf("quota.GetQuota failed. Error: %s", err)
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
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetTokenInfoById")

		return nil, cErr
	}
	return abi.GetTokenByID(sd, tokenId)
}

func (c *chain) GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressAsset)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
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
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetStakeAmountByPage")

		return nil, nil, cErr
	}
	return abi.GetStakeListByPage(sd, lastKey, count)
}

func (c *chain) GetDexFundsByPage(snapshotHash types.Hash, lastAddress types.Address, count int) ([]*dex.Fund, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetStakeAmountByPage")
		return nil, cErr
	}
	return dex.GetUserFundsByPage(sd, lastAddress, count)
}

func (c *chain) GetDexStakeListByPage(snapshotHash types.Hash, lastKey []byte, count int) ([]*dex.DelegateStakeInfo, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetDexStakeListByPage")
		return nil, nil, cErr
	}
	return dex.GetStakeListByPage(sd, lastKey, count)
}

func (c *chain) GetDexFundByAddress(snapshotHash types.Hash, address types.Address) (*dex.Fund, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetDexFundByAddress")
		return nil, cErr
	}
	if v, err1 := sd.GetValue(dex.GetFundKey(address)); err1 != nil {
		return nil, err1
	} else {
		if len(v) > 0 {
			fund := &dex.Fund{}
			err2 := fund.DeSerialize(v)
			return fund, err2
		} else {
			return nil, nil
		}
	}
}

func (c *chain) GetDexFundStakeForMiningV1ListByPage(snapshotHash types.Hash, lastKey []byte, count int) ([]*types.Address, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetDexFundStakeForMiningV1ListByPage")
		return nil, nil, cErr
	}
	return dex.GetStakeForMiningV1ByPage(sd, lastKey, count)
}

func (c *chain) GetDexFundStakeForMiningV2ListByPage(snapshotHash types.Hash, lastKey []byte, count int) ([]*types.Address, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewStorageDatabase failed")
		c.log.Error(cErr.Error(), "method", "GetDexFundStakeForMiningV2ListByPage")
		return nil, nil, cErr
	}
	return dex.GetStakeForMiningV2ByPage(sd, lastKey, count)
}
