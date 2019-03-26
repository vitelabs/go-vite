package types

type Quota struct {
	current uint64
	total   uint64
	avg     uint64
}

func NewQuota(total, used, avg uint64) Quota {
	if total > used {
		return Quota{total - used, total, avg}
	} else {
		return Quota{0, total, avg}
	}
}

// Total quota of a single account in 75 snapshot blocks
func (q *Quota) Total() uint64 {
	return q.total
}

// Current quota of a single account
func (q *Quota) Current() uint64 {
	return q.current
}

// Quota used of a single account in past 74 snapshot blocks and unconfirmed account blocks
func (q *Quota) Used() uint64 {
	return q.total - q.current
}

func (q *Quota) Avg() uint64 {
	return q.avg
}
