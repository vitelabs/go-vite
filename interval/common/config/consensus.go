package config

import (
	"strconv"

	"github.com/pkg/errors"
)

type Consensus struct {
	Interval int `yaml:"interval"`
	MemCnt   int `yaml:"memCnt"`
}

func (ch *Consensus) Check(cfg *Base) error {
	if ch.Interval <= 0 {
		errors.New("interval must gt 0, interval:" + strconv.Itoa(ch.Interval))
	}

	if ch.MemCnt <= 0 {
		errors.New("MemCnt must gt 0, MemCnt:" + strconv.Itoa(ch.MemCnt))
	}
	return nil
}
