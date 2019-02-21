package filters

import "github.com/vitelabs/go-vite/vite"

type EventSystem struct {
	vite *vite.Vite
	// TODO channels
}

func NewEventSystem(v *vite.Vite) *EventSystem {
	// TODO
	return &EventSystem{vite: v}
}
