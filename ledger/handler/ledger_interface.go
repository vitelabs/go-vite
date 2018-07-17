package handler

import "github.com/vitelabs/go-vite/protocols"

type Vite interface {
	Pm () *protocols.ProtocolManager
}