package config

import (
	"strconv"

	"github.com/pkg/errors"
)

type Consensus struct {
	Interval int `yaml:"interval"`
	MemCnt   int `yaml:"memCnt"`
}

func (self *Consensus) Check(cfg *Base) error {
	if self.Interval <= 0 {
		errors.New("interval must gt 0, interval:" + strconv.Itoa(self.Interval))
	}

	if self.MemCnt <= 0 {
		errors.New("MemCnt must gt 0, MemCnt:" + strconv.Itoa(self.MemCnt))
	}
	return nil
}
