package net

// use to filter redundant fetch

type Filter interface {
}

type filter struct {
	chain *skeleton
}

func newFilter() *filter {
	return &filter{}
}
