package onroad_test

import (
	"testing"
	"github.com/vitelabs/go-vite/onroad"
)

func TestManager_InitAndStartWork(t *testing.T) {
	onroad.NewManager(new(onroad.Vite), "12")

}
