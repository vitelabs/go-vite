package pool

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/log15"

	"github.com/hashicorp/golang-lru"
)

type Blacklist interface {
	Add(key types.Hash)
	AddAddTimeout(key types.Hash, duration time.Duration)
	Exists(key types.Hash) bool
	Remove(key types.Hash)
}

func NewBlacklist() (Blacklist, error) {
	cache, err := lru.New(10 * 10000)
	if err != nil {
		return nil, err
	}
	return &blacklist{cache: cache, defaultTimeout: 0, log: log15.New("module", "pool/blacklist")}, nil
}

type timeout struct {
	timeoutT *time.Time
}

func (self *timeout) reset(duration time.Duration) *timeout {
	if duration <= 0 {
		self.timeoutT = nil
	} else {
		t := time.Now().Add(duration)
		self.timeoutT = &t
	}

	return self
}
func (self *timeout) isTimeout() bool {
	if self.timeoutT == nil {
		return false
	}
	return self.timeoutT.Before(time.Now())
}

type blacklist struct {
	cache          *lru.Cache
	defaultTimeout time.Duration
	log            log15.Logger
}

func (self *blacklist) Add(key types.Hash) {
	self.AddAddTimeout(key, self.defaultTimeout)
}

func (self *blacklist) AddAddTimeout(key types.Hash, duration time.Duration) {
	value, ok := self.cache.Get(key)
	if ok {
		value.(*timeout).reset(duration)
	} else {
		self.cache.Add(key, (&timeout{}).reset(duration))
	}
}

func (self *blacklist) Exists(key types.Hash) bool {
	value, ok := self.cache.Get(key)
	if ok {
		return !value.(*timeout).isTimeout()
	}
	return false
}

func (self *blacklist) Remove(key types.Hash) {
	self.cache.Remove(key)
}
