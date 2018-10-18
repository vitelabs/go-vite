package common

import "github.com/vitelabs/go-vite/log15"

var glog = log15.New("module", "error")

func Go(fn func()) {
	go wrap(fn)
}

func wrap(fn func()) {
	defer catch()
	fn()
}

func catch() {
	if err := recover(); err != nil {
		glog.Error("panic", "err", err)
		panic(err)
	}
}
