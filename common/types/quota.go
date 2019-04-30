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
	unconfirmed                 uint64
}

func NewQuota(utps, current, avg, unconfirmed uint64) Quota {
	return Quota{current, utps, avg, unconfirmed}
}

func (q *Quota) PledgeQuotaPerSnapshotBlock() uint64 {
	return q.pledgeQuotaPerSnapshotBlock
}

// Current quota of a single account
func (q *Quota) Current() uint64 {
	return q.current
}

// Quota used of a single account in past 74 snapshot blocks and unconfirmed account blocks
func (q *Quota) SnapshotCurrent() uint64 {
	return q.current - q.unconfirmed
}

func (q *Quota) Avg() uint64 {
	return q.avg
}
