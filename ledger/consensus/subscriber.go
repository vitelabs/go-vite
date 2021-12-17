package consensus

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/v2/common/types"
)

type subscribeTrigger interface {
	triggerEvent(gid types.Gid, fn func(*subscribeEvent))
	triggerProducerEvent(gid types.Gid, fn func(*producerSubscribeEvent))
}

type consensusSubscriber struct {
	firstSub  *sync.Map
	secondSub *sync.Map
}

func newConsensusSubscriber() *consensusSubscriber {
	return &consensusSubscriber{firstSub: &sync.Map{}, secondSub: &sync.Map{}}
}

func (cs *consensusSubscriber) selected(gid types.Gid) *sync.Map {
	if gid == types.SNAPSHOT_GID {
		return cs.firstSub
	} else if gid == types.DELEGATE_GID {
		return cs.secondSub
	}
	return nil
}

func (cs *consensusSubscriber) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event)) {
	sub := cs.selected(gid)
	sub.Store(id, &subscribeEvent{addr: addr, fn: fn, gid: gid})
}
func (cs *consensusSubscriber) UnSubscribe(gid types.Gid, id string) {
	cs.selected(gid).Delete(id)
}

func (cs *consensusSubscriber) SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent)) {
	sub := cs.selected(gid)
	sub.Store(id, &producerSubscribeEvent{fn: fn, gid: gid})
}

func (cs consensusSubscriber) triggerEvent(gid types.Gid, fn func(*subscribeEvent)) {
	cs.selected(gid).Range(func(k, v interface{}) bool {
		switch t := v.(type) {
		case *subscribeEvent:
			fn(t)
		}
		return true
	})
}

func (cs consensusSubscriber) triggerProducerEvent(gid types.Gid, fn func(*producerSubscribeEvent)) {
	cs.selected(gid).Range(func(k, v interface{}) bool {
		switch t := v.(type) {
		case *producerSubscribeEvent:
			fn(t)
		}
		return true
	})
}

func (cs consensusSubscriber) TriggerMineEvent(addr types.Address) error {
	return errors.New("not supported")
}
