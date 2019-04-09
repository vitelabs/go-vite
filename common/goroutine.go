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
		var e error
		switch t := err.(type) {
		case error:
			e = errors.WithStack(t)
		case string:
			e = errors.New(t)
		default:
			e = errors.Errorf("unknown type %+v", err)
		}

		glog.Error("panic", "err", err, "withstack", e)
		fmt.Printf("%+v", e)
		panic(err)
	}
}
