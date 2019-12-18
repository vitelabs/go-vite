package api

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/vite"
)

type Health struct {
	vite *vite.Vite
}

func NewHealthApi(vite *vite.Vite) *Health {
	return &Health{vite: vite}
}

func (h *Health) Health() error {
	sb := h.vite.Chain().GetLatestSnapshotBlock()
	if sb == nil {
		return errors.New("check node height failed, sb nil")
	}
	if err := checkSnapshotValid(sb); err != nil {
		return errors.New("check node height failed, height invalid")
	}
	return nil
}
