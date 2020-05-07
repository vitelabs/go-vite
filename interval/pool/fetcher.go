package pool

import (
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/syncer"
)

type fetcher struct {
	// if address == "", snapshot fetcher
	// else account fetcher
	address string
	fetcher syncer.Fetcher
}

func NewFetcher(address string, f syncer.Fetcher) *fetcher {
	self := &fetcher{}
	self.address = address
	self.fetcher = f
	return self
}

func (f *fetcher) fetch(hashHeight common.HashHeight, prevCnt uint64) {
	f.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: f.address})
}

func (f *fetcher) fetchRequest(request face.FetchRequest) {
	f.fetcher.Fetch(request)
}
func (f *fetcher) fetchReqs(reqs []face.FetchRequest) {
	for _, r := range reqs {
		f.fetcher.Fetch(r)
	}
}
