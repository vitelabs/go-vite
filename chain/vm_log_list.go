package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetVmLogList(logListHash *types.Hash) (ledger.VmLogList, error) {
	if logListHash == nil {
		return nil, nil
	}

	logList, err := c.stateDB.GetVmLogList(logListHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetVmLogList failed, error is %s, logListHash is %s", err, logListHash))
		c.log.Error(cErr.Error(), "method", "GetVmLogList")
		return nil, cErr
	}
	return logList, nil
}
