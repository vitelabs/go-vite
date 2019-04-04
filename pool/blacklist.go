package pool

import (
	"time"

	"github.com/hashicorp/golang-lru"
)

type Blacklist interface {
	Add(key interface{})
	AddAddTimeout(key interface{}, duration time.Duration)
	Exists(key interface{}) bool
	Remove(key interface{})
}

func NewBlacklist() (Blacklist, error) {
	cache, err := lru.New(10 * 10000)
	if err != nil {
		return nil, err
	}
	return &blacklist{cache: cache, defaultTimeout: 0}, nil
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
}

func (self *blacklist) Add(key interface{}) {
	self.AddAddTimeout(key, self.defaultTimeout)
}

func (self *blacklist) AddAddTimeout(key interface{}, duration time.Duration) {
	value, ok := self.cache.Get(key)
	if ok {
		value.(*timeout).reset(duration)
	} else {
		self.cache.Add(key, (&timeout{}).reset(duration))
	}
}

func (self *blacklist) Exists(key interface{}) bool {
	value, ok := self.cache.Get(key)
	if ok {
		return !value.(*timeout).isTimeout()
	}
	return false
}

func (self *blacklist) Remove(key interface{}) {
	self.cache.Remove(key)
}
