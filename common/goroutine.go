package common

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
)

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
		fmt.Printf("%+v", errors.WithStack(err.(error)))
		panic(err)
	}
}
