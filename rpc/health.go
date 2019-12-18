package rpc

import (
	"net/url"
	"strings"
)

/*
var TimeAlarmLimit time.Duration = 5 * time.Minute

type HealthCheck struct {
	client *Client
	log    log15.Logger
}

func NewHealthCheckClient(endpointUrl string) (*HealthCheck, error) {
	c, err := DialHTTP(endpointUrl)
	if err != nil {
		return nil, err
	}
	r := &HealthCheck{client: c, log: log15.New("module", "health")}
	return r, nil
}

func (h *HealthCheck) HealthCheck(w http.ResponseWriter, req *http.Request) {
	snapshotBlock := &api.SnapshotBlock{}
	err := h.client.Call(snapshotBlock, "ledger_getLatestSnapshotBlock", nil)
	if err != nil || snapshotBlock == nil || !checkTime(snapshotBlock.Timestamp) {
		h.log.Info("[failure]check node height")
		http.Error(w, "check node height failed", http.StatusServiceUnavailable)
		return
	}
	h.log.Info("[success]check node height")
	w.WriteHeader(http.StatusOK)
}

func checkTime(snapshotTime int64) bool {
	nowTime := time.Now()
	sTime := time.Unix(int64(snapshotTime), 0)
	if nowTime.Before(sTime.Add(-TimeAlarmLimit)) || nowTime.After(sTime.Add(TimeAlarmLimit)) {
		return false
	}
	return true
}*/

func isHealthCheckRouter(url *url.URL) bool {
	return strings.EqualFold(url.Path, "/health")
}
