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

func (self *fetcher) fetch(hashHeight common.HashHeight, prevCnt uint64) {
	self.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: self.address})
}

func (self *fetcher) fetchRequest(request face.FetchRequest) {
	self.fetcher.Fetch(request)
}
func (self *fetcher) fetchReqs(reqs []face.FetchRequest) {
	for _, r := range reqs {
		self.fetcher.Fetch(r)
	}
}
