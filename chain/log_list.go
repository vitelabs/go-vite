package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error) {
	vmLogList, err := c.chainDb.Ac.GetVmLogList(logListHash)
	if err != nil {
		c.log.Error("GetVmLogList failed, error is "+err.Error(), "method", "GetVmLogList")
		return nil, err
	}

	return vmLogList, nil
}
