package common

import "github.com/vitelabs/go-vite/log15"

var glog = log15.New("module", "error")

func Go(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Error("panic", "err", err)
				panic(err)
			}
		}()
		fn()
	}()
}
