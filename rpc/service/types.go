package service

const (
	MAX_QUERY_COUNT = 50
)

type Paging struct {
	index int   // Page index
	num int 	// Number of pages
	count int   // Count of elements per page
}

type AccountBlockQuery struct {
	Paging
}

type SnapshotBlockQuery struct {
	Paging
}

type TokenQuery struct {
	Paging
} 