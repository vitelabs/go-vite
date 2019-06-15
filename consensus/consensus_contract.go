package consensus

import (
	"sync"
	"time"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type contractsCs struct {
	rw *chainRw

	// all consensus group for contract
	contracts   map[types.Gid]*contractDposCs
	contractsMu sync.Mutex

	log log15.Logger
}

func newContractCs(rw *chainRw, log log15.Logger) *contractsCs {
	cs := &contractsCs{}
	cs.rw = rw
	cs.log = log.New("gid", "contracts")
	cs.contracts = make(map[types.Gid]*contractDposCs)
	return cs
}

func (contract *contractsCs) LoadGid(gid types.Gid) error {
	result, err := contract.reloadGid(gid)
	if err != nil {
		return err
	}
	if result == nil {
		return errors.Errorf("load contract consensus group[%s] fail.", gid)
	}
	return nil
}

func (contract *contractsCs) ElectionTime(gid types.Gid, t time.Time) (*electionResult, error) {
	result, err := contract.getOrLoadGid(gid)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.Errorf("can't load contract group for gid:%s, t:%s", gid, t)
	}
	return result.electionTime(t)
}

func (contract *contractsCs) ElectionIndex(gid types.Gid, index uint64) (*electionResult, error) {
	result, err := contract.getOrLoadGid(gid)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.Errorf("can't load contract group for gid:%s, index:%d", gid, index)
	}
	return result.ElectionIndex(index)
}

func (contract *contractsCs) getOrLoadGid(gid types.Gid) (*contractDposCs, error) {
	cs := contract.getForGid(gid)
	if cs == nil {
		tmp, err := contract.reloadGid(gid)
		if err != nil {
			return nil, err
		}
		return tmp, nil
	}
	return cs, nil
}

func (contract *contractsCs) reloadGid(gid types.Gid) (*contractDposCs, error) {
	contract.contractsMu.Lock()
	defer contract.contractsMu.Unlock()
	info, err := contract.rw.GetMemberInfo(gid)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, errors.Errorf("can't load consensus gid:%s", gid)
	}
	cs := newContractDposCs(info, contract.rw, contract.log)
	contract.contracts[gid] = cs
	return cs, nil
}

func (contract *contractsCs) getForGid(gid types.Gid) *contractDposCs {
	contract.contractsMu.Lock()
	defer contract.contractsMu.Unlock()

	cs, ok := contract.contracts[gid]
	if ok {
		return cs
	}
	return nil
}
