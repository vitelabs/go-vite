package onroad_test

import (
	"github.com/vitelabs/go-vite/onroad"
)

func newManager() {
	manager := onroad.NewManager(new(testVite))
	manager.Init()
}
