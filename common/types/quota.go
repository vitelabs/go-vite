package types

type QuotaInfo struct {
	BlockCount     uint64
	QuotaTotal     uint64
	QuotaUsedTotal uint64
}

type Quota struct {
	current                     uint64
	pledgeQuotaPerSnapshotBlock uint64
	avg                         uint64
	snapshotCurrent             uint64
	blocked                     bool
}

func NewQuota(pledgeQuota, current, avg, snapshotCurrent uint64, blocked bool) Quota {
	return Quota{current, pledgeQuota, avg, snapshotCurrent, blocked}
}

func (q *Quota) PledgeQuotaPerSnapshotBlock() uint64 {
	return q.pledgeQuotaPerSnapshotBlock
}

// Current quota of a single account
func (q *Quota) Current() uint64 {
	return q.current
}

// Available quota in current snapshot block, excluding unconfirmed blocks
func (q *Quota) SnapshotCurrent() uint64 {
	return q.snapshotCurrent
}

func (q *Quota) Avg() uint64 {
	return q.avg
}

func (q *Quota) Blocked() bool {
	return q.blocked
}
