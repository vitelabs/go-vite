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
	// todo
	return cs
}

func (self *contractsCs) LoadGid(gid types.Gid) error {
	result, err := self.reloadGid(gid)
	if err != nil {
		return err
	}
	if result == nil {
		return errors.Errorf("load contract consensus group[%s] fail.", gid)
	}
	return nil
}

func (self *contractsCs) ElectionTime(gid types.Gid, t time.Time) (*electionResult, error) {
	result, err := self.getOrLoadGid(gid)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.Errorf("can't load contract group for gid:%s, t:%s", gid, t)
	}
	return result.electionTime(t)
}

func (self *contractsCs) ElectionIndex(gid types.Gid, index uint64) (*electionResult, error) {
	result, err := self.getOrLoadGid(gid)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.Errorf("can't load contract group for gid:%s, index:%s", gid, index)
	}
	return result.electionIndex(index)
}

func (self *contractsCs) getOrLoadGid(gid types.Gid) (*contractDposCs, error) {
	cs := self.getForGid(gid)
	if cs == nil {
		tmp, err := self.reloadGid(gid)
		if err != nil {
			return nil, err
		}
		return tmp, nil
	}
	return cs, nil
}

func (self *contractsCs) reloadGid(gid types.Gid) (*contractDposCs, error) {
	self.contractsMu.Lock()
	defer self.contractsMu.Unlock()
	info, err := self.rw.GetMemberInfo(gid)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, errors.Errorf("can't load consensus gid:%s", gid)
	}
	cs := newContractDposCs(info, self.rw, self.log)
	self.contracts[gid] = cs
	return cs, nil
}

func (self *contractsCs) getForGid(gid types.Gid) *contractDposCs {
	self.contractsMu.Lock()
	defer self.contractsMu.Unlock()

	cs, ok := self.contracts[gid]
	if ok {
		return cs
	}
	return nil
}
