package unconfirmed

import "github.com/vitelabs/go-vite/unconfirmed/worker"

func StartUp() {
	var vite worker.Vite
	uManager := NewManager(vite)

}
