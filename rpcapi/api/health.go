package api

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/vite"
	"time"
)

const InvalidSnapshotMinutes = 3

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
	nowTime := time.Now()
	if nowTime.After(sb.Timestamp.Add(InvalidSnapshotMinutes * time.Minute)) {
		return errors.New("check node height failed, height invalid")
	}
	return nil
}
