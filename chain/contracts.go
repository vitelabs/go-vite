package chain

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
	"time"
)

func (c *chain) GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error) {
	monitorTags := []string{"chain", "GetContractGidByAccountBlock"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if block == nil {
		return nil, nil
	}

	if block.Height == 1 {
		if types.IsBuiltinContractAddrInUse(block.AccountAddress) {
			return &types.DELEGATE_GID, nil
		}

		fromBlock, err := c.GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil {
			return nil, err
		}

		return c.chainDb.Ac.GetContractGidFromSendCreateBlock(fromBlock)
	}
	return c.GetContractGid(&block.AccountAddress)
}

// TODO cache
func (c *chain) GetContractGid(addr *types.Address) (*types.Gid, error) {
	monitorTags := []string{"chain", "GetContractGid"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if addr == nil {
		return nil, nil
	}

	if types.IsBuiltinContractAddrInUse(*addr) {
		return &types.DELEGATE_GID, nil
	}

	account, getAccountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if getAccountErr != nil {

		c.log.Error("Query account failed. Error is "+getAccountErr.Error(), "method", "GetContractGid")

		return nil, getAccountErr
	}
	if account == nil {
		return nil, nil
	}

	gid, err := c.chainDb.Ac.GetContractGid(account.AccountId)

	if err != nil {
		c.log.Error("GetContractGid failed, error is "+err.Error(), "method", "GetContractGid")
		return nil, err
	}

	return gid, nil
}

func (c *chain) GetPledgeQuotas(snapshotHash types.Hash, beneficialList []types.Address) (map[types.Address]types.Quota, error) {
	monitorTags := []string{"chain", "GetPledgeQuotas"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())
	// TODO
	return nil, nil

	/*pledgeDb, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuotas")
		return nil, err
	}
	quotas := make(map[types.Address]uint64)
	for _, addr := range beneficialList {
		balanceDb, err := vm_context.NewVmContext(c, &snapshotHash, nil, &addr)
		if err != nil {
			c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuotas")
			return nil, err
		}
		pledgeAmount := abi.GetPledgeBeneficialAmount(pledgeDb, addr)
		quotas[addr], err = quota.GetPledgeQuota(balanceDb, addr, pledgeAmount)
		if err != nil {
			return nil, err
		}
	}
	return quotas, nil*/
}
func (c *chain) GetPledgeQuota(snapshotHash types.Hash, beneficial types.Address) (types.Quota, error) {
	monitorTags := []string{"chain", "GetPledgeQuota"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())
	// TODO
	return types.Quota{}, nil
	/*vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &beneficial)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuota")
		return 0, err
	}
	pledgeAmount := abi.GetPledgeBeneficialAmount(vmContext, beneficial)
	return quota.GetPledgeQuota(vmContext, beneficial, pledgeAmount)*/
}

func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	// TODO
	return nil, nil
	/*monitorTags := []string{"chain", "GetRegisterList"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetCandidateList")
		return nil, err
	}
	return abi.GetCandidateList(vmContext, gid, nil), nil*/
}

func (c *chain) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	// TODO
	return nil, nil
	/*monitorTags := []string{"chain", "GetVoteMap"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetVoteList")
		return nil, err
	}
	return abi.GetVoteList(vmContext, gid, nil), nil*/
}

func (c *chain) GetPledgeAmount(snapshotHash types.Hash, beneficial types.Address) (*big.Int, error) {
	// TODO
	return nil, nil
	/*monitorTags := []string{"chain", "GetPledgeAmount"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeBeneficialAmount")
		return nil, err
	}
	return abi.GetPledgeBeneficialAmount(vmContext, beneficial), nil*/
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	// TODO
	return nil, nil
	/*monitorTags := []string{"chain", "GetConsensusGroupList"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetActiveConsensusGroupList")
		return nil, err
	}
	return abi.GetActiveConsensusGroupList(vmContext, nil), nil*/
}

func (c *chain) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) {
	monitorTags := []string{"chain", "GetBalanceList"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetBalanceList")
		return nil, err
	}

	balanceList := make(map[types.Address]*big.Int)
	for _, addr := range addressList {
		balanceList[addr] = vmContext.GetBalance(&addr, &tokenTypeId)
	}
	return balanceList, nil
}

func (c *chain) GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error) {
	// TODO
	return nil, nil
	/*monitorTags := []string{"chain", "GetTokenInfoById"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	vmContext, err := vm_context.NewVmContext(c, nil, nil, &types.AddressMintage)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetTokenInfoById")
		return nil, err
	}
	return abi.GetTokenById(vmContext, *tokenId), nil*/
}
