package p2p

import "time"

type flow struct {
	threshold int
	overflow  int
	duration  time.Duration
}
