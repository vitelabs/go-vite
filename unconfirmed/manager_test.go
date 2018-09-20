package unconfirmed_test

import (
	"testing"
	"github.com/vitelabs/go-vite/unconfirmed"
)

func TestManager_InitAndStartWork(t *testing.T) {
	unconfirmed.NewManager(new(unconfirmed.Vite), "12")

}
