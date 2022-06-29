package consensus

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (cs *consensus) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	cs.snapshot.verifyProducerAndSeed(header)
	return cs.snapshot.VerifyProducer(header.Producer(), *header.Timestamp)
}

func (cs *consensus) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	var result []*ledger.AccountBlock
	for k, v := range abs {
		blocks, err := cs.VerifyABsProducerByGid(k, v)
		if err != nil {
			return nil, err
		}
		result = append(result, blocks...)
	}
	return result, nil
}

func (cs *consensus) VerifyABsProducerByGid(gid types.Gid, blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	tel, err := cs.contracts.getOrLoadGid(gid)

	if err != nil {
		return nil, err
	}
	if tel == nil {
		return nil, errors.New("consensus group not exist")
	}
	return tel.verifyAccountsProducer(blocks)
}

func (cs *consensus) VerifyAccountProducer(accountBlock *ledger.AccountBlock) (bool, error) {
	gid, err := cs.rw.getGid(accountBlock)
	if err != nil {
		return false, err
	}
	tel, err := cs.contracts.getOrLoadGid(*gid)

	if err != nil {
		return false, err
	}
	if tel == nil {
		return false, errors.New("consensus group not exist")
	}
	return tel.VerifyAccountProducer(accountBlock)
}

func (cs *consensus) ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error) {
	// load from dpos wrapper
	reader, err := cs.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return nil, 0, err
	}

	// cal votes
	eResult, err := reader.ElectionIndex(index)
	if err != nil {
		return nil, 0, err
	}

	voteTime := cs.snapshot.GenProofTime(index)
	var result []*Event
	for _, p := range eResult.Plans {
		e := newConsensusEvent(eResult, p, gid, voteTime)
		result = append(result, &e)
	}
	return result, uint64(eResult.Index), nil
}

func (cs *consensus) VoteTimeToIndex(gid types.Gid, t2 time.Time) (uint64, error) {
	// load from dpos wrapper
	reader, err := cs.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return 0, err
	}
	return reader.Time2Index(t2), nil
}

func (cs *consensus) VoteIndexToTime(gid types.Gid, i uint64) (*time.Time, *time.Time, error) {
	// load from dpos wrapper
	reader, err := cs.dposWrapper.getDposConsensus(gid)
	if err != nil {
		return nil, nil, errors.Errorf("consensus group[%s] not exist", gid)
	}

	st, et := reader.Index2Time(i)
	return &st, &et, nil
}

func (cs *consensus) Init(cfg *ConsensusCfg) error {
	if !cs.PreInit() {
		panic("pre init fail.")
	}
	defer cs.PostInit()
	if cfg == nil {
		cfg = DefaultCfg()
	}
	cs.ConsensusCfg = cfg

	cs.rw.init(cs.snapshot)

	cs.tg = newTrigger(cs.rollback)
	err := cs.contracts.LoadGid(types.DELEGATE_GID)

	if err != nil {
		panic(err)
	}

	return nil
}

func (cs *consensus) Start() {
	cs.PreStart()
	defer cs.PostStart()
	cs.closed = make(chan struct{})
	cs.ctx, cs.cancelFn = context.WithCancel(context.Background())

	common.Go(func() {
		cs.wg.Add(1)
		defer cs.wg.Done()
		cs.tg.update(cs.ctx, types.SNAPSHOT_GID, cs.snapshot, cs.subscribeTrigger)
	})

	reader, err := cs.dposWrapper.getDposConsensus(types.DELEGATE_GID)
	if err != nil {
		panic(err)
	}

	common.Go(func() {
		cs.wg.Add(1)
		defer cs.wg.Done()
		cs.tg.update(cs.ctx, types.DELEGATE_GID, reader, cs.subscribeTrigger)
	})

	cs.rw.Start()
	//cs.rw.rw.Register(cs)
}

func (cs *consensus) Stop() {
	cs.PreStop()
	defer cs.PostStop()
	//cs.rw.rw.UnRegister(cs)
	cs.rw.Stop()
	cs.cancelFn()
	close(cs.closed)
	cs.wg.Wait()
}
